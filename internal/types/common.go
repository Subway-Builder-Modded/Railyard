package types

type Status string

const (
	ResponseSuccess Status = "success"
	ResponseError   Status = "error"
	ResponseWarn    Status = "warn"
)

type GenericResponse struct {
	Status  Status `json:"status"`
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

type AssetType string

const (
	AssetTypeMap AssetType = "map"
	AssetTypeMod AssetType = "mod"
)

func IsValidAssetType(assetType AssetType) bool {
	switch assetType {
	case AssetTypeMap, AssetTypeMod:
		return true
	default:
		return false
	}
}

type Version string
