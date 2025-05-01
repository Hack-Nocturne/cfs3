package vars

const (
	KB_SIZE                    = 1_024
	API_BASE_URL               = "https://api.cloudflare.com/client/v4"
	MAX_ASSET_COUNT            = 20_000
	MAX_ASSET_SIZE             = 25 * KB_SIZE * KB_SIZE
	BULK_UPLOAD_CONCURRENCY    = 6
	MAX_BUCKET_FILE_COUNT      = 2_500
	MAX_BUCKET_SIZE            = 72 * KB_SIZE * KB_SIZE // 72MB * 4/3 (base64) = 96MB (max size of a single request: 100MB)
	MAX_CHECK_MISSING_ATTEMPTS = 4
	MAX_UPLOAD_ATTEMPTS        = 8
	MAX_UPLOAD_GATEWAY_ERRORS  = 4
)
