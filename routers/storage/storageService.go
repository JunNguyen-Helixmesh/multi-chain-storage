package storage

import (
	"context"
	"mime/multipart"
	"multi-chain-storage/blockchain/browsersync/scanlockpayment/polygon"
	"multi-chain-storage/common"
	"multi-chain-storage/common/constants"
	"multi-chain-storage/common/utils"
	"multi-chain-storage/config"
	"multi-chain-storage/database"
	"multi-chain-storage/logs"
	"multi-chain-storage/models"
	"multi-chain-storage/on-chain/goBind"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	common2 "github.com/ethereum/go-ethereum/common"
	clientmodel "github.com/filswan/go-swan-client/model"
	"github.com/filswan/go-swan-client/subcommand"
	"github.com/filswan/go-swan-lib/client/ipfs"
	libmodel "github.com/filswan/go-swan-lib/model"
	libutils "github.com/filswan/go-swan-lib/utils"
	"github.com/gin-gonic/gin"
)

var createDirMutext = &sync.Mutex{}

func SaveFileAndCreateCarAndUploadToIPFSAndSaveDb(c *gin.Context, srcFile *multipart.FileHeader, duration, userId int, walletAddress string, fileType int) (string, string, int, error) {
	temDirDeal := config.GetConfig().SwanTask.DirDeal

	logs.GetLogger().Info("temp dir is ", temDirDeal)
	needPay := 0
	homedir, err := os.UserHomeDir()
	if err != nil {
		logs.GetLogger().Error("Cannot get home directory.")
		return "", "", needPay, err
	}
	temDirDeal = filepath.Join(homedir, temDirDeal[2:])

	err = libutils.CreateDir(temDirDeal)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", "", needPay, err
	}

	createDirMutext.Lock()
	timeStr := time.Now().Format("20060102_150405")
	tempDirDealTemp := filepath.Join(temDirDeal, timeStr)
	temDirDeal = tempDirDealTemp
	i := 0
	for libutils.IsDirExists(temDirDeal) {
		temDirDeal = tempDirDealTemp + "_" + strconv.Itoa(i)
		i++
	}
	err = libutils.CreateDir(temDirDeal)
	if err != nil {
		logs.GetLogger().Error(err)
		createDirMutext.Unlock()
		return "", "", needPay, err
	}
	createDirMutext.Unlock()

	srcDir := filepath.Join(temDirDeal, "src")
	carDir := filepath.Join(temDirDeal, "car")
	err = libutils.CreateDir(srcDir)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", "", needPay, err
	}

	err = libutils.CreateDir(carDir)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", "", needPay, err
	}

	srcFilepath := filepath.Join(srcDir, srcFile.Filename)
	logs.GetLogger().Info("save your file to ", srcFilepath)
	err = c.SaveUploadedFile(srcFile, srcFilepath)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", "", needPay, err
	}
	logs.GetLogger().Info("car files will be created in ", carDir)

	confCar := clientmodel.ConfCar{
		LotusClientApiUrl:      config.GetConfig().Lotus.ApiUrl,
		LotusClientAccessToken: config.GetConfig().Lotus.AccessToken,
		InputDir:               srcDir,
		OutputDir:              carDir,
	}
	fileList, err := subcommand.CreateCarFiles(&confCar)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", "", needPay, err
	}
	logs.GetLogger().Info("car files created in ", carDir, ", payload_cid=", fileList[0].DataCid)

	uploadUrl := utils.UrlJoin(config.GetConfig().IpfsServer.UploadUrl, "api/v0/add?stream-channels=true&pin=true")
	ipfsFileHash, err := ipfs.IpfsUploadFileByWebApi(uploadUrl, srcFilepath)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", "", needPay, err
	}

	filePathInIpfs := config.GetConfig().IpfsServer.DownloadUrlPrefix + constants.IPFS_URL_PREFIX_BEFORE_HASH + *ipfsFileHash
	lockPaymentList, err := models.FindEventLockPayment(&models.EventLockPayment{PayloadCid: fileList[0].DataCid}, "create_at desc", "10", "0")
	if err != nil {
		logs.GetLogger().Error(err)
		return "", "", needPay, err
	}

	sourceAndDealFileList, err := GetSourceFileAndDealFileInfoByPayloadCid(fileList[0].DataCid)
	if err != nil {
		logs.GetLogger().Error(err)
		return "", "", needPay, err
	}

	if len(sourceAndDealFileList) > 0 {
		if sourceAndDealFileList[0].WalletAddress == walletAddress {
			if len(lockPaymentList) > 0 {
				needPay = 1
				return fileList[0].DataCid, sourceAndDealFileList[0].IpfsUrl, needPay, nil
			} else {
				needPay = 2
				return fileList[0].DataCid, sourceAndDealFileList[0].IpfsUrl, needPay, nil
			}
		} else {
			if len(lockPaymentList) > 0 {
				needPay = 3
				sourceFile, err := saveSourceFileToDB(srcFile, srcFilepath, userId, filePathInIpfs, walletAddress, fileType)
				if err != nil {
					logs.GetLogger().Error(err)
					return "", "", needPay, err
				}
				filepMap := new(models.SourceFileDealFileMap)
				filepMap.SourceFileId = sourceFile.ID
				filepMap.DealFileId = sourceAndDealFileList[0].ID
				filepMap.FileIndex = 0
				nowTime := strconv.FormatInt(utils.GetEpochInMillis(), 10)
				filepMap.CreateAt = nowTime
				filepMap.UpdateAt = nowTime
				err = database.SaveOne(filepMap)
				if err != nil {
					logs.GetLogger().Error(err)
					return "", "", needPay, err
				}
				return fileList[0].DataCid, sourceAndDealFileList[0].IpfsUrl, needPay, nil
			} else {
				needPay = 4
				sourceFile, err := saveSourceFileToDB(srcFile, srcFilepath, userId, filePathInIpfs, walletAddress, fileType)
				if err != nil {
					logs.GetLogger().Error(err)
					return "", "", needPay, err
				}
				err = saveDealFileAndMapRelation(fileList, sourceFile, duration)
				if err != nil {
					logs.GetLogger().Error(err)
					return "", "", needPay, err
				}
				return fileList[0].DataCid, filePathInIpfs, needPay, nil
			}
		}
	} else {
		sourceFile, err := saveSourceFileToDB(srcFile, srcFilepath, userId, filePathInIpfs, walletAddress, fileType)
		if err != nil {
			logs.GetLogger().Error(err)
			return "", "", needPay, err
		}
		err = saveDealFileAndMapRelation(fileList, sourceFile, duration)
		if err != nil {
			logs.GetLogger().Error(err)
			return "", "", needPay, err
		}
		return fileList[0].DataCid, filePathInIpfs, needPay, nil
	}
}

func saveSourceFileToDB(srcFile *multipart.FileHeader, srcFilepath string, userId int, filePathInIpfs, walletAddress string, fileType int) (*models.SourceFile, error) {
	sourceFile := new(models.SourceFile)
	sourceFile.FileName = srcFile.Filename
	sourceFile.FileSize = strconv.FormatInt(srcFile.Size, 10)
	sourceFile.ResourceUri = srcFilepath
	sourceFile.CreateAt = strconv.FormatInt(utils.GetEpochInMillis(), 10)
	//todo userid
	//sourceFile.UserId = userId
	sourceFile.IpfsUrl = filePathInIpfs
	sourceFile.PinStatus = constants.IPFS_File_PINNED_STATUS
	sourceFile.WalletAddress = walletAddress
	sourceFile.FileType = strconv.Itoa(fileType)
	err := database.SaveOne(sourceFile)
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	return sourceFile, nil
}

func GetSourceFileAndDealFileInfoByPayloadCid(payloadCid string) ([]*SourceFileAndDealFileInfo, error) {
	sql := "select s.wallet_address,s.ipfs_url,s.file_name,d.id,d.payload_cid,d.deal_cid,d.deal_id,d.lock_payment_status,s.create_at from source_file s,source_file_deal_file_map m,deal_file d " +
		" where s.id = m.source_file_id and m.deal_file_id = d.id and d.payload_cid='" + payloadCid + "'"
	var results []*SourceFileAndDealFileInfo
	err := database.GetDB().Raw(sql).Order("create_at desc").Limit(10).Offset(0).Order("create_at desc").Scan(&results).Error
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	return results, nil
}

/*
func CreateTask(){
	taskDataset := config.GetConfig().SwanTask.CuratedDataset
	taskDescription := config.GetConfig().SwanTask.Description
	startEpochIntervalHours := config.GetConfig().SwanTask.StartEpochHours
	startEpoch := libutils.GetCurrentEpoch() + (startEpochIntervalHours+1)*libconstants.EPOCH_PER_HOUR
	confTask := &clientmodel.ConfTask{
		SwanApiUrl:                 config.GetConfig().SwanApi.ApiUrl,
		SwanToken:                  jwtToken,
		PublicDeal:                 true,
		BidMode:                    libconstants.TASK_BID_MODE_AUTO,
		VerifiedDeal:               config.GetConfig().SwanTask.VerifiedDeal,
		OfflineMode:                false,
		FastRetrieval:              config.GetConfig().SwanTask.FastRetrieval,
		MaxPrice:                   config.GetConfig().SwanTask.MaxPrice,
		StorageServerType:          libconstants.STORAGE_SERVER_TYPE_IPFS_SERVER,
		WebServerDownloadUrlPrefix: config.GetConfig().IpfsServer.DownloadUrlPrefix,
		ExpireDays:                 config.GetConfig().SwanTask.ExpireDays,
		OutputDir:                  carDir,
		InputDir:                   carDir,
		//TaskName:                   taskName,
		MinerFid:                "",
		Dataset:                 taskDataset,
		Description:             taskDescription,
		StartEpochIntervalHours: startEpochIntervalHours,
		StartEpoch:              startEpoch,
		SourceId:                constants.SOURCE_ID_OF_PAYMENT,
		Duration:                duration,
	}

	_, fileInfoList, _, err := subcommand.CreateTask(confTask, nil)
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	if len(fileInfoList) > 0 {
		err = saveDealFileAndMapRelation(fileInfoList, sourceFile)
		if err != nil {
			logs.GetLogger().Error(err)
			return nil, err
		}
	}

	logs.GetLogger().Info("task created")
	return fileInfoList, nil
}*/

func saveDealFileAndMapRelation(fileInfoList []*libmodel.FileDesc, sourceFile *models.SourceFile, duration int) error {
	currentTime := utils.GetEpochInMillis()
	dealFile := new(models.DealFile)
	dealFile.CarFileName = fileInfoList[0].CarFileName
	dealFile.CarFilePath = fileInfoList[0].CarFilePath
	dealFile.CarFileSize = fileInfoList[0].CarFileSize
	dealFile.CarMd5 = fileInfoList[0].CarFileMd5
	dealFile.PayloadCid = fileInfoList[0].DataCid
	dealFile.PieceCid = fileInfoList[0].PieceCid
	dealFile.SourceFilePath = sourceFile.ResourceUri
	dealFile.DealCid = fileInfoList[0].DealCid
	dealFile.CreateAt = strconv.FormatInt(currentTime, 10)
	dealFile.UpdateAt = strconv.FormatInt(currentTime, 10)
	dealFile.Duration = duration
	dealFile.LockPaymentStatus = constants.LOCK_PAYMENT_STATUS_WAITING
	dealFile.IsDeleted = utils.GetBoolPointer(false)
	err := database.SaveOne(dealFile)
	if err != nil {
		logs.GetLogger().Error(err)
		return err
	}

	filepMap := new(models.SourceFileDealFileMap)
	filepMap.SourceFileId = sourceFile.ID
	filepMap.DealFileId = dealFile.ID
	filepMap.FileIndex = 0
	filepMap.CreateAt = strconv.FormatInt(currentTime, 10)
	filepMap.UpdateAt = strconv.FormatInt(currentTime, 10)
	err = database.SaveOne(filepMap)
	if err != nil {
		logs.GetLogger().Error(err)
		return err
	}
	return nil
}

func GetSourceFileAndDealFileInfo(limit, offset string, walletAddress string, payloadCid, fileName string, paymentStatus string, mintStatus string) ([]*SourceFileAndDealFileInfoExtend, error) {
	sql := "select s.file_name,s.file_size,s.pin_status,s.create_at,df.miner_fid,df.payload_cid,df.deal_cid,df.deal_id,df.piece_cid,df.deal_status,df.lock_payment_status as status,df.duration, evpm.locked_fee as locked_fee, s.nft_tx_hash, s.token_id, s.mint_address from source_file s " +
		" inner join source_file_deal_file_map sfdfm on s.id = sfdfm.source_file_id" +
		" inner join deal_file df on sfdfm.deal_file_id = df.id and wallet_address='" + walletAddress + "' "
	if strings.Trim(payloadCid, " ") != "" {
		sql = sql + " and df.payload_cid='" + payloadCid + "'"
	}
	if strings.Trim(fileName, " ") != "" {
		sql = sql + " and s.file_name like '%" + fileName + "%'"
	}
	if strings.Trim(paymentStatus, " ") != "" {
		switch paymentStatus {
		case "0": // Unpaid
			sql = sql + " and df.lock_payment_status not in ('" + constants.LOCK_PAYMENT_STATUS_PAID + "','" + constants.LOCK_PAYMENT_STATUS_SUCCESS + "','" + constants.LOCK_PAYMENT_STATUS_REFUNDED + "','" + constants.LOCK_PAYMENT_STATUS_REFUNDING + "','" + constants.LOCK_PAYMENT_STATUS_PROCESSING + "')"
		case "1": // Paid
			sql = sql + " and df.lock_payment_status in ('" + constants.LOCK_PAYMENT_STATUS_PAID + "','" + constants.LOCK_PAYMENT_STATUS_SUCCESS + "','" + constants.LOCK_PAYMENT_STATUS_PROCESSING + "')"
		case "2": // Refunding
			sql = sql + " and df.lock_payment_status in ('" + constants.LOCK_PAYMENT_STATUS_REFUNDING + "')"
		case "3": // Refunded
			sql = sql + " and df.lock_payment_status in ('" + constants.LOCK_PAYMENT_STATUS_REFUNDED + "')"
		default:

		}
	}

	if strings.Trim(mintStatus, " ") != "" {
		switch mintStatus {
		case "0": // Not mint
			sql = sql + " and TRIM(IFNULL(s.nft_tx_hash,'')) = '' "
		case "1": // Mint
			sql = sql + " and TRIM(IFNULL(s.nft_tx_hash,'')) <> '' "
		default:

		}
	}
	sql = sql + " left outer join event_lock_payment evpm on evpm.payload_cid = df.payload_cid"
	sql = sql + " where IFNULL(s.file_type,'0') <>'1'"
	var results []*SourceFileAndDealFileInfoExtend
	err := database.GetDB().Raw(sql).Order("create_at desc").Limit(limit).Offset(offset).Scan(&results).Error
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	return results, nil
}

func GetSourceFileAndDealFileInfoCount(walletAddress string, payloadCid, fileName string, paymentStatus string, mintStatus string) (int64, error) {
	sql := "select count(1) as total_record from  source_file s " +
		" inner join source_file_deal_file_map sfdfm on s.id = sfdfm.source_file_id" +
		" inner join deal_file df on sfdfm.deal_file_id = df.id" +
		" where s.wallet_address='" + walletAddress + "'"
	if strings.Trim(payloadCid, " ") != "" {
		sql = sql + " and df.payload_cid='" + payloadCid + "'"
	}
	if strings.Trim(fileName, " ") != "" {
		sql = sql + " and s.file_name like '%" + fileName + "%'"
	}
	if strings.Trim(paymentStatus, " ") != "" {
		switch paymentStatus {
		case "0": // Unpaid
			sql = sql + " and df.lock_payment_status not in ('" + constants.LOCK_PAYMENT_STATUS_PAID + "','" + constants.LOCK_PAYMENT_STATUS_SUCCESS + "','" + constants.LOCK_PAYMENT_STATUS_REFUNDED + "','" + constants.LOCK_PAYMENT_STATUS_REFUNDING + "','" + constants.LOCK_PAYMENT_STATUS_PROCESSING + "')"
		case "1": // Paid
			sql = sql + " and df.lock_payment_status in ('" + constants.LOCK_PAYMENT_STATUS_PAID + "','" + constants.LOCK_PAYMENT_STATUS_SUCCESS + "','" + constants.LOCK_PAYMENT_STATUS_PROCESSING + "')"
		case "2": // Refunding
			sql = sql + " and df.lock_payment_status in ('" + constants.LOCK_PAYMENT_STATUS_REFUNDING + "')"
		case "3": // Refunded
			sql = sql + " and df.lock_payment_status in ('" + constants.LOCK_PAYMENT_STATUS_REFUNDED + "')"
		default:

		}
	}

	if strings.Trim(mintStatus, " ") != "" {
		switch mintStatus {
		case "0": // Not mint
			sql = sql + " and TRIM(IFNULL(s.nft_tx_hash,'')) = '' "
		case "1": // Mint
			sql = sql + " and TRIM(IFNULL(s.nft_tx_hash,'')) <> '' "
		default:

		}
	}
	sql = sql + " and IFNULL(s.file_type,'0') <>'1'"

	var recordCount common.RecordCount
	err := database.GetDB().Raw(sql).Scan(&recordCount).Error
	if err != nil {
		logs.GetLogger().Error(err)
		return 0, err
	}
	return recordCount.TotalRecord, nil
}

func GetDealListThanGreaterDealID(dealId int64, offset, limit int) ([]*DaoDealResult, error) {
	whereCondition := "deal_id > " + strconv.FormatInt(dealId, 10)
	var results []*DaoDealResult
	err := database.GetDB().Table("deal_file").Where(whereCondition).Offset(offset).Limit(limit).Order("create_at").Scan(&results).Error
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	return results, nil
}

func GetDaoSignatureInfoByDealId(dealId int64) ([]*DaoSignResult, error) {
	whereCondition := "deal_id = " + strconv.FormatInt(dealId, 10)
	var results []*DaoSignResult
	err := database.GetDB().Table("event_dao_signature").Where(whereCondition).Offset(0).Limit(constants.DEFAULT_SELECT_LIMIT).Order("block_time desc").Scan(&results).Error
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	return results, nil
}

func GetLockFoundInfoByPayloadCid(payloadCid string) (*LockFound, error) {
	lockEventList, err := models.FindEventLockPayment(&models.EventLockPayment{PayloadCid: payloadCid}, "", "10", "0")
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err

	}
	lockFound := new(LockFound)
	if len(lockEventList) > 0 {
		lockFound.PayloadCid = lockEventList[0].PayloadCid
		lockFound.LockedFee = lockEventList[0].LockedFee
		lockFound.CreateAt = lockEventList[0].CreateAt
	}
	return lockFound, nil
}

func GetShoulBeSignDealListFromDB() ([]*DealForDaoSignResult, error) {
	finalSql := "select deal_id,deal_cid,piece_cid,payload_cid,cost,verified,miner_fid,duration,client_wallet_address,create_at from deal_file where deal_id not in  ( " +
		"     select  deal_id from dao_fetched_deal ) " +
		" and deal_id > 0 order by create_at desc"
	var dealForDaoSignResultList []*DealForDaoSignResult
	err := database.GetDB().Raw(finalSql).Scan(&dealForDaoSignResultList).Limit(0).Offset(constants.DEFAULT_SELECT_LIMIT).Error
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err

	}
	return dealForDaoSignResultList, nil
}

func GetDaoSignEventByDealId(dealId int64) ([]*DaoInfoResult, error) {
	if dealId == 0 {
		dealId = constants.FILE_BLOCK_NUMBER_MAX
	}
	finalSql := " select * from( " +
		" (select dao_name, dao_address,order_index from dao_info order by order_index asc)) as d  left  join " +
		" (select deal_id,tx_hash,dao_pass_time,if(deal_id > 0,1,2) as status,dao_address as dao_address_event,payload_cid  from event_dao_signature where deal_id = " + strconv.FormatInt(dealId, 10) + " ) as a " +
		" on d.dao_address=a.dao_address_event"

	var daoInfoResult []*DaoInfoResult
	err := database.GetDB().Raw(finalSql).Scan(&daoInfoResult).Limit(0).Offset(constants.DEFAULT_SELECT_LIMIT).Error
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err
	}
	return daoInfoResult, nil
}

func GetThreshHold() (uint8, error) {
	daoAddress := common2.HexToAddress(polygon.GetConfig().PolygonMainnetNode.DaoSwanOracleAddress)
	client := polygon.WebConn.ConnWeb

	pk := os.Getenv("privateKeyOnPolygon")
	if strings.HasPrefix(strings.ToLower(pk), "0x") {
		pk = pk[2:]
	}

	callOpts := new(bind.CallOpts)
	callOpts.From = daoAddress
	callOpts.Context = context.Background()

	daoOracleContractInstance, err := goBind.NewFilswanOracle(daoAddress, client)
	if err != nil {
		logs.GetLogger().Error(err)
		return 0, err
	}

	threshHold, err := daoOracleContractInstance.GetThreshold(callOpts)
	if err != nil {
		logs.GetLogger().Error(err)
		return 0, err
	}
	logs.GetLogger().Info("dao threshHold is : ", threshHold)
	return threshHold, nil
}

func GetDealFileInfoByPayloadCid(payloadCid string) ([]*models.DealFile, error) {
	dealFileList, err := models.FindDealFileList(&models.DealFile{PayloadCid: payloadCid}, "", "10", "0")
	if err != nil {
		logs.GetLogger().Error(err)
		return nil, err

	}
	return dealFileList, nil
}
