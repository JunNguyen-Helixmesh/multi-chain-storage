package constants

const (
	PAGE_SIZE_DEFAULT_VALUE = 10

	HTTP_STATUS_SUCCESS = "success"
	HTTP_STATUS_ERROR   = "error"

	SOURCE_ID_MCS = 4

	CAR_FILE_STATUS_TASK_CREATED      = "TaskCreated"
	CAR_FILE_STATUS_DEAL_SENT         = "DealSent"
	CAR_FILE_STATUS_DEAL_SENT_FAILED  = "DealSentFailed"
	CAR_FILE_STATUS_DEAL_SEND_EXPIRED = "DealSentExpired"
	CAR_FILE_STATUS_SUCCESS           = "Success"

	SOURCE_FILE_UPLOAD_STATUS_PENDING      = "Pending"
	SOURCE_FILE_UPLOAD_STATUS_PROCESSING   = "Processing"
	SOURCE_FILE_UPLOAD_STATUS_PAID         = "Paid"
	SOURCE_FILE_UPLOAD_STATUS_TASK_CREATED = "TaskCreated"
	SOURCE_FILE_UPLOAD_STATUS_UNLOCKING    = "Unlocking"
	SOURCE_FILE_UPLOAD_STATUS_SUCCESS      = "Success"
	SOURCE_FILE_UPLOAD_STATUS_REFUNDABLE   = "Refundable"
	SOURCE_FILE_UPLOAD_STATUS_REFUNDED     = "Refunded"

	OFFLINE_DEAL_STATUS_CREATED = "Created"
	OFFLINE_DEAL_STATUS_SUCCESS = "Success"
	OFFLINE_DEAL_STATUS_ACTIVE  = "Active"
	OFFLINE_DEAL_STATUS_FAILED  = "Failed"

	ON_CHAIN_DEAL_STATUS_ACTIVE = "StorageDealActive"
	ON_CHAIN_DEAL_STATUS_ERROR  = "StorageDealError"

	IPFS_URL_PREFIX_BEFORE_HASH = "/ipfs/"
	IPFS_File_PINNED_STATUS     = "Pinned"

	DAO_SIGNATURE_STATUS_SUCCESS = "Success"
	DAO_SIGNATURE_STATUS_FAILED  = "Failed"

	DURATION_DAYS_DEFAULT = 525

	SOURCE_FILE_TYPE_NORMAL = 0
	SOURCE_FILE_TYPE_MINT   = 1

	BYTES_1GB     = 1024 * 1024 * 1024
	EPOCH_PER_DAY = 24 * 60 * 2

	PRIVATE_KEY_ON_POLYGON = "privateKeyOnPolygon"

	TOKEN_USDC_NAME   = "USDC"
	TOKEN_USDC_ADRESS = "USDC_ADDRESS"

	WALLET_TYPE_META_MASK = 0
	WALLET_TYPE_FILE_COIN = 1

	NETWORK_NAME_POLYGON = "polygon"

	SYSTEM_PARAM_PAYMENT_CONTRACT_ADDRESS  = "PAYMENT_CONTRACT_ADDRESS"
	SYSTEM_PARAM_PAYMENT_RECIPIENT_ADDRESS = "PAYMENT_RECIPIENT_ADDRESS"
	SYSTEM_PARAM_DAO_CONTRACT_ADDRESS      = "DAO_CONTRACT_ADDRESS"
	SYSTEM_PARAM_MINT_CONTRACT_ADDRESS     = "MINT_CONTRACT_ADDRESS"
	SYSTEM_PARAM_GAS_LIMIT                 = "GAS_LIMIT"
	SYSTEM_PARAM_LOCK_TIME                 = "LOCK_TIME"
	SYSTEM_PARAM_PAY_MULTIPLY_FACTOR       = "PAY_MULTIPLY_FACTOR"
)
