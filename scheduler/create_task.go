package scheduler

import (
	"math/big"
	"path/filepath"
	"payment-bridge/blockchain/browsersync/scanlockpayment/polygon"
	"payment-bridge/common/constants"
	"payment-bridge/config"
	"payment-bridge/database"
	"payment-bridge/models"
	"payment-bridge/routers/billing"

	"github.com/filswan/go-swan-lib/logs"

	"github.com/filswan/go-swan-client/command"
	libconstants "github.com/filswan/go-swan-lib/constants"
	libutils "github.com/filswan/go-swan-lib/utils"
	"github.com/shopspring/decimal"
)

func CheckIfHaveLockPayment(payloadCid string) ([]*models.EventLockPayment, error) {
	polygonEventList, err := models.FindEventLockPayment(&models.EventLockPayment{PayloadCid: payloadCid}, "id desc", "", "0")
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	return polygonEventList, nil
}

func createTask() error {
	whereCondition := "lower(lock_payment_status)=lower('" + constants.LOCK_PAYMENT_STATUS_PROCESSING + "') and task_uuid = '' and is_deleted=false "
	dealList, err := models.FindDealFileList(whereCondition, "create_at desc", "50", "0")
	if err != nil {
		logs.GetLogger().Error(err)
		return err
	}
	for _, deal := range dealList {
		totalLockFee, err := models.GetTotalLockFeeByCarPayloadCid(deal.PayloadCid)
		if err != nil {
			logs.GetLogger().Error(err)
			continue
		}

		lockedFee := totalLockFee.IntPart()

		fileCoinPriceInUsdc, err := billing.GetWfilPriceFromSushiPrice(polygon.WebConn.ConnWeb, "1")
		if err != nil {
			logs.GetLogger().Error(err)
			return err
		}
		maxPrice, err := GetMaxPriceForCreateTask(fileCoinPriceInUsdc, lockedFee, deal.Duration, deal.CarFileSize)
		if err != nil {
			logs.GetLogger().Error(err)
			continue
		}
		logs.GetLogger().Println("payload cid ", deal.PayloadCid, " max price is ", maxPrice)

		if maxPrice.Cmp(config.GetConfig().SwanTask.MaxPrice) < 0 {
			*maxPrice = config.GetConfig().SwanTask.MaxPrice
		}

		cmdUpload := command.CmdUpload{
			StorageServerType:           libconstants.STORAGE_SERVER_TYPE_IPFS_SERVER,
			IpfsServerDownloadUrlPrefix: config.GetConfig().IpfsServer.DownloadUrlPrefix,
			IpfsServerUploadUrlPrefix:   config.GetConfig().IpfsServer.UploadUrlPrefix,
			OutputDir:                   filepath.Dir(deal.CarFilePath),
			InputDir:                    filepath.Dir(deal.CarFilePath),
		}

		_, err = cmdUpload.UploadCarFiles()
		if err != nil {
			logs.GetLogger().Error(err)
			return err
		}
		logs.GetLogger().Info("car files uploaded")

		taskDataset := config.GetConfig().SwanTask.CuratedDataset
		taskDescription := config.GetConfig().SwanTask.Description
		startEpochIntervalHours := config.GetConfig().SwanTask.StartEpochHours

		cmdTask := command.CmdTask{
			SwanApiUrl:                 config.GetConfig().SwanApi.ApiUrl,
			SwanToken:                  "",
			SwanApiKey:                 config.GetConfig().SwanApi.ApiKey,
			SwanAccessToken:            config.GetConfig().SwanApi.AccessToken,
			BidMode:                    libconstants.TASK_BID_MODE_AUTO,
			VerifiedDeal:               config.GetConfig().SwanTask.VerifiedDeal,
			OfflineMode:                false,
			FastRetrieval:              config.GetConfig().SwanTask.FastRetrieval,
			MaxPrice:                   *maxPrice,
			StorageServerType:          libconstants.STORAGE_SERVER_TYPE_IPFS_SERVER,
			WebServerDownloadUrlPrefix: config.GetConfig().IpfsServer.DownloadUrlPrefix,
			ExpireDays:                 config.GetConfig().SwanTask.ExpireDays,
			OutputDir:                  filepath.Dir(deal.CarFilePath),
			InputDir:                   filepath.Dir(deal.CarFilePath),
			Dataset:                    taskDataset,
			Description:                taskDescription,
			StartEpochHours:            startEpochIntervalHours,
			SourceId:                   constants.SOURCE_ID_OF_PAYMENT,
			Duration:                   deal.Duration,
			MaxAutoBidCopyNumber:       5,
		}

		_, fileDescs, _, err := cmdTask.CreateTask(nil)
		if err != nil {
			logs.GetLogger().Error(err)
			return err
		}
		if len(fileDescs) > 0 {
			taskUuid := fileDescs[0].Uuid
			logs.GetLogger().Info("task created, uuid=", taskUuid)
			deal.TaskUuid = taskUuid
			err := database.SaveOne(deal)
			if err != nil {
				logs.GetLogger().Error(err)
				return err
			}
			return nil
		}
	}
	return nil
}

func GetMaxPriceForCreateTask(rate *big.Int, lockedFee int64, duration int, carFileSize int64) (*decimal.Decimal, error) {
	_, sectorSize := libutils.CalculatePieceSize(carFileSize)
	lockedFeeDecimal := decimal.NewFromInt(lockedFee)
	lockedFeeInFileCoin := lockedFeeDecimal.Div(decimal.NewFromFloat(constants.LOTUS_PRICE_MULTIPLE_1E18)).Div(decimal.NewFromInt(rate.Int64()))
	maxPrice := lockedFeeInFileCoin.Div(decimal.NewFromFloat(sectorSize).Div(decimal.NewFromInt(10204 * 1024 * 1024))).Div(decimal.NewFromInt(int64(duration)))
	return &maxPrice, nil
}
