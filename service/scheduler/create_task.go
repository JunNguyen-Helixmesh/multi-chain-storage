package scheduler

import (
	"fmt"
	"math/big"
	"multi-chain-storage/common/constants"
	"multi-chain-storage/common/utils"
	"multi-chain-storage/config"
	"multi-chain-storage/database"
	"multi-chain-storage/models"
	"multi-chain-storage/on-chain/client"
	"os"
	"path/filepath"
	"time"

	"github.com/filswan/go-swan-client/command"
	libconstants "github.com/filswan/go-swan-lib/constants"
	"github.com/filswan/go-swan-lib/logs"
	libmodel "github.com/filswan/go-swan-lib/model"
	libutils "github.com/filswan/go-swan-lib/utils"
	"github.com/shopspring/decimal"
)

func CreateTask() error {
	err := CheckSourceFilesPaid()
	if err != nil {
		logs.GetLogger().Error(err)
	}

	for {
		numSrcFiles, err := createTask()
		if err != nil {
			logs.GetLogger().Error(err)
			return err
		}

		if numSrcFiles == nil || *numSrcFiles == 0 {
			logs.GetLogger().Info("0 source file created to car file")
			return nil
		}

		logs.GetLogger().Info(*numSrcFiles, " source file(s) created to car file")
	}

}

func createTask() (*int, error) {
	srcFiles, err := models.GetSourceFileUploadsNeed2Car()
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}

	if len(srcFiles) == 0 {
		logs.GetLogger().Info("0 source file upload to be created to car file")
		return nil, nil
	}

	currentTimeStr := time.Now().Format("2006-01-02T15:04:05")
	carSrcDir := filepath.Join(carDir, "src_"+currentTimeStr)
	carDestDir := filepath.Join(carDir, "car_"+currentTimeStr)

	err = libutils.CreateDir(carSrcDir)
	if err != nil {
		logs.GetLogger().Error("creating dir:", carSrcDir, " failed,", err)
		return nil, err
	}

	totalSize := int64(0)
	currentUtcMilliSec := utils.GetCurrentUtcSecond()
	createdTimeMin := currentUtcMilliSec
	var maxPrice *decimal.Decimal

	fileCoinPriceInUsdc, err := client.GetWfilPriceFromSushiPrice("1")
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}

	fileSizeMin := config.GetConfig().SwanTask.MinFileSize
	var srcFiles2Merged []*models.SourceFileUploadsNeed2Car
	for _, srcFile := range srcFiles {
		srcFilepathTemp := filepath.Join(carSrcDir, filepath.Base(srcFile.ResourceUri))

		bytesCopied, err := libutils.CopyFile(srcFile.ResourceUri, srcFilepathTemp)
		if err != nil {
			logs.GetLogger().Error(err)
			os.Remove(srcFilepathTemp)
			continue
		}

		maxPriceTemp, err := getMaxPrice(srcFile.FileSize, srcFile.LockedFee, fileCoinPriceInUsdc)
		if err != nil {
			logs.GetLogger().Error(err)
			os.Remove(srcFilepathTemp)
			continue
		}

		if maxPrice == nil {
			maxPrice = maxPriceTemp
		} else if maxPrice.Cmp(*maxPriceTemp) > 0 {
			*maxPrice = *maxPriceTemp
		}

		totalSize = totalSize + bytesCopied

		if srcFile.CreateAt < createdTimeMin {
			createdTimeMin = srcFile.CreateAt
		}

		srcFiles2Merged = append(srcFiles2Merged, srcFile)

		if totalSize >= fileSizeMin {
			logs.GetLogger().Info("total size is:", totalSize, ", ", len(srcFiles2Merged), " files to be created to car file")
			break
		}
	}

	if totalSize == 0 {
		os.RemoveAll(carSrcDir)
		logs.GetLogger().Info("0 source file to be created to car file")
		return nil, nil
	}

	passedMilliSec := currentUtcMilliSec - createdTimeMin
	createAnyway := false
	if passedMilliSec >= 24*60*60*1000 {
		createAnyway = true
	}

	if !createAnyway && totalSize < fileSizeMin {
		err := fmt.Errorf("source file size:%d is less than min file size:%d", totalSize, fileSizeMin)
		logs.GetLogger().Error(err)
		os.RemoveAll(carSrcDir)
		return nil, err
	}

	err = libutils.CreateDir(carDestDir)
	if err != nil {
		logs.GetLogger().Error("creating dir:", carDestDir, " failed,", err)
		os.RemoveAll(carSrcDir)
		return nil, err
	}

	fileDesc, err := createTask4SrcFiles(carSrcDir, carDestDir, *maxPrice)
	if err != nil {
		logs.GetLogger().Error(err)
		os.RemoveAll(carSrcDir)
		os.RemoveAll(carDestDir)
		return nil, err
	}

	err = saveCarInfo2DB(fileDesc, srcFiles2Merged, *maxPrice)
	if err != nil {
		os.RemoveAll(carSrcDir)
		os.RemoveAll(carDestDir)
		logs.GetLogger().Error(err)
		return nil, err
	}

	err = os.RemoveAll(carSrcDir)
	if err != nil {
		logs.GetLogger().Error(err)
	}

	numSrcFiles := len(srcFiles2Merged)
	return &numSrcFiles, nil
}

func getMaxPrice(fileSize int64, lockedFee decimal.Decimal, rate *big.Int) (*decimal.Decimal, error) {
	_, sectorSize := libutils.CalculatePieceSize(fileSize)

	lockedFeeInFileCoin := lockedFee.Div(decimal.NewFromFloat(libconstants.LOTUS_PRICE_MULTIPLE_1E18)).Div(decimal.NewFromInt(rate.Int64()))

	durationEpoch := decimal.NewFromInt(constants.DURATION_DAYS_DEFAULT * constants.EPOCH_PER_DAY)
	sectorSizeGB := decimal.NewFromFloat(sectorSize).Div(decimal.NewFromInt(constants.BYTES_1GB))

	maxPrice := lockedFeeInFileCoin.Div(sectorSizeGB).Div(durationEpoch)

	confMaxPrice := config.GetConfig().SwanTask.MaxPrice

	if maxPrice.Cmp(confMaxPrice) > 0 {
		maxPrice = confMaxPrice
	}

	return &maxPrice, nil
}

func createTask4SrcFiles(srcDir, carDir string, maxPrice decimal.Decimal) (*libmodel.FileDesc, error) {
	cmdIpfsCar := &command.CmdIpfsCar{
		LotusClientApiUrl:         config.GetConfig().Lotus.ClientApiUrl,
		LotusClientAccessToken:    config.GetConfig().Lotus.ClientAccessToken,
		InputDir:                  srcDir,
		OutputDir:                 carDir,
		GenerateMd5:               false,
		IpfsServerUploadUrlPrefix: config.GetConfig().IpfsServer.UploadUrlPrefix,
	}

	_, err := cmdIpfsCar.CreateIpfsCarFiles()
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	logs.GetLogger().Info("car files created to ", carDir, " from ", srcDir)

	cmdUpload := command.CmdUpload{
		StorageServerType:           libconstants.STORAGE_SERVER_TYPE_IPFS_SERVER,
		IpfsServerDownloadUrlPrefix: config.GetConfig().IpfsServer.DownloadUrlPrefix,
		IpfsServerUploadUrlPrefix:   config.GetConfig().IpfsServer.UploadUrlPrefix,
		InputDir:                    carDir,
	}

	_, err = cmdUpload.UploadCarFiles()
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	logs.GetLogger().Info("car files uploaded to ipfs from ", carDir)

	cmdTask := command.CmdTask{
		SwanApiUrl:                 config.GetConfig().SwanApi.ApiUrl,
		SwanToken:                  "",
		SwanApiKey:                 config.GetConfig().SwanApi.ApiKey,
		SwanAccessToken:            config.GetConfig().SwanApi.AccessToken,
		LotusClientApiUrl:          config.GetConfig().Lotus.ClientApiUrl,
		BidMode:                    libconstants.TASK_BID_MODE_AUTO,
		VerifiedDeal:               config.GetConfig().SwanTask.VerifiedDeal,
		OfflineMode:                false,
		FastRetrieval:              config.GetConfig().SwanTask.FastRetrieval,
		MaxPrice:                   maxPrice,
		StorageServerType:          libconstants.STORAGE_SERVER_TYPE_IPFS_SERVER,
		WebServerDownloadUrlPrefix: config.GetConfig().IpfsServer.DownloadUrlPrefix,
		ExpireDays:                 config.GetConfig().SwanTask.ExpireDays,
		InputDir:                   carDir,
		OutputDir:                  carDir,
		Dataset:                    config.GetConfig().SwanTask.CuratedDataset,
		Description:                config.GetConfig().SwanTask.Description,
		StartEpochHours:            config.GetConfig().SwanTask.StartEpochHours,
		SourceId:                   constants.SOURCE_ID_OF_PAYMENT,
		Duration:                   constants.DURATION_DAYS_DEFAULT * 24 * 60 * 2,
		MaxAutoBidCopyNumber:       5,
	}

	_, fileDescs, _, err := cmdTask.CreateTask(nil)
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}

	fileDesc := fileDescs[0]

	logs.GetLogger().Info("task created for car files in ", carDir, ",payload_cid=", fileDesc.PayloadCid)

	return fileDesc, nil
}

func saveCarInfo2DB(fileDesc *libmodel.FileDesc, srcFiles []*models.SourceFileUploadsNeed2Car, maxPrice decimal.Decimal) error {
	db := database.GetDBTransaction()
	currentUtcSecond := utils.GetCurrentUtcSecond()
	dealFile := models.CarFile{
		CarFileName: fileDesc.CarFileName,
		CarFilePath: fileDesc.CarFilePath,
		CarFileSize: fileDesc.CarFileSize,
		PayloadCid:  fileDesc.PayloadCid,
		PieceCid:    fileDesc.PieceCid,
		CreateAt:    currentUtcSecond,
		UpdateAt:    currentUtcSecond,
		Duration:    constants.DURATION_DAYS_DEFAULT,
		Status:      constants.CAR_FILE_STATUS_TASK_CREATED,
		MaxPrice:    maxPrice,
		TaskUuid:    fileDesc.Uuid,
	}

	err := database.SaveOneInTransaction(db, &dealFile)
	if err != nil {
		db.Rollback()
		logs.GetLogger().Error(err)
		return err
	}

	for _, srcFile := range srcFiles {
		filepMap := models.CarFileSource{
			CarFileId:          dealFile.ID,
			SourceFileUploadId: srcFile.SourceFileUploadId,
			CreateAt:           currentUtcSecond,
		}
		err = database.SaveOneInTransaction(db, &filepMap)
		if err != nil {
			db.Rollback()
			logs.GetLogger().Error(err)
			return err
		}

		sql := "update source_file_upload set status=?,update_at=? where id=?"

		params := []interface{}{}
		params = append(params, constants.SOURCE_FILE_UPLOAD_STATUS_TASK_CREATED)
		params = append(params, currentUtcSecond)
		params = append(params, srcFile.SourceFileUploadId)

		err = db.Exec(sql, params...).Error
		if err != nil {
			db.Rollback()
			logs.GetLogger().Error(err)
			return err
		}
	}

	err = db.Commit().Error
	if err != nil {
		logs.GetLogger().Error(err)
		return err
	}

	return nil
}

func CheckSourceFilesPaid() error {
	srcFileUploads, err := models.GetSourceFileUploadsByFileTypeStatus(constants.SOURCE_FILE_TYPE_NORMAL, constants.SOURCE_FILE_UPLOAD_STATUS_CREATED)
	if err != nil {
		logs.GetLogger().Error(err)
		return err
	}

	for _, srcFileUpload := range srcFileUploads {
		err = models.CreateTransaction(srcFileUpload.Id, "")
		if err != nil {
			logs.GetLogger().Error(err)
			continue
		}
	}

	return nil
}
