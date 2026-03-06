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

// MissingFilesError is returned when required files are missing from an archive.
type MissingFilesError struct {
	Files []string
}

func (e *MissingFilesError) Error() string {
	return "Missing required files: " + joinStrings(e.Files, ", ")
}

// MapAlreadyExistsError is returned when a map code conflicts with an existing map.
type MapAlreadyExistsError struct {
	MapCode string
}

func (e *MapAlreadyExistsError) Error() string {
	return "Map with code '" + e.MapCode + "' has already been installed or would overwrite a vanilla map."
}

func joinStrings(s []string, sep string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += sep
		}
		result += v
	}
	return result
}
