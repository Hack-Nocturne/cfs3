package types

// UploadPayloadFile represents a file upload payload.
type UploadPayloadFile struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Metadata struct {
		ContentType string `json:"contentType"`
	} `json:"metadata"`
	Base64 bool `json:"base64"`
}

// FileContainer holds file metadata.
type FileContainer struct {
	Path        string
	ContentType string
	SizeInBytes int64
	Hash        string
}

// UploadArgs holds parameters for the upload function.
type UploadArgs struct {
	FileMap     map[string]FileContainer
	Jwt         *string
	AccountId   string
	ProjectName string
	SkipCaching bool
}

// Represents the response from the upload API.
type UploadResponse struct {
	SuccessfullKeyCount int      `json:"successful_key_count"`
	UnsuccessfulKeys    []string `json:"unsuccessful_keys"`
}
