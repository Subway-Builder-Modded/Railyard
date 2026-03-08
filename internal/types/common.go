package types

import (
	"strings"

	"golang.org/x/mod/semver"
)

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

func IsValidSemverVersion(version Version) bool {
	value := strings.TrimSpace(string(version))
	if value == "" {
		return false
	}
	if strings.ContainsAny(value, "-+") {
		return false
	}
	if !strings.HasPrefix(value, "v") {
		value = "v" + value
	}
	if !semver.IsValid(value) {
		return false
	}

	core := value[1:]
	if idx := strings.IndexAny(core, "-+"); idx >= 0 {
		core = core[:idx]
	}
	return strings.Count(core, ".") == 2
}

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
