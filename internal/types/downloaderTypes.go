package types

type GenericResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type DownloadTempResponse struct {
	GenericResponse
	Path string `json:"path,omitempty"`
}

type MapExtractResponse struct {
	GenericResponse
	Config ConfigData `json:"config,omitempty"`
}
