package types

// FileContainer holds file metadata.
type FileContainer struct { // ! This is done to prevent import cycles between utils and worker packages.
	Path        string
	ContentType string
	SizeInBytes int64
	Hash        string
}
