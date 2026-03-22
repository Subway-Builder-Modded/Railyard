package types

import (
	"io"
	"regexp"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

type Status string

const (
	ResponseSuccess Status = "success"
	ResponseError   Status = "error"
	ResponseWarn    Status = "warn"
)

const RequestTimeout = 15 * time.Second
const RequestUserAgent = "Railyard-Desktop-App"
const GitHubAPIBaseURL = "https://api.github.com"
const GitHubTokenDocsURL = "https://subwaybuildermodded.com/railyard/docs/latest/players/github-token"

type APIErrorType string

const (
	APIErrorTypeAuthInvalidToken APIErrorType = "api_auth_invalid_token"
	APIErrorTypeRateLimited      APIErrorType = "api_rate_limited"
	APIErrorTypeForbidden        APIErrorType = "api_forbidden"
	APIErrorTypeNotFound         APIErrorType = "api_not_found"
	APIErrorTypeBadRequest       APIErrorType = "api_bad_request"
	APIErrorTypeUpstream5xx      APIErrorType = "api_upstream_5xx"
	APIErrorTypeStatus           APIErrorType = "api_status_error"
	APIErrorTypeTimeout          APIErrorType = "api_timeout"
	APIErrorTypeNetwork          APIErrorType = "api_network_error"
	APIErrorTypeFetch            APIErrorType = "api_fetch_error"
)

type APIErrorSource string

const (
	APIErrorSourceGitHub APIErrorSource = "Github"
)

type GenericResponse struct {
	Status         Status         `json:"status"`
	Message        string         `json:"message"`
	APIErrorType   APIErrorType   `json:"apiErrorType,omitempty"`
	APIErrorSource APIErrorSource `json:"apiErrorSource,omitempty"`
	APIStatusCode  int            `json:"apiStatusCode,omitempty"`
}

type DownloadTempResponse struct {
	GenericResponse
	Path string `json:"path,omitempty"`
}

type AssetConflict struct {
	ExistingAssetID   string    `json:"existingAssetId"`
	ExistingAssetType AssetType `json:"existingAssetType"`
	ExistingVersion   string    `json:"existingVersion"`
	ExistingIsLocal   bool      `json:"existingIsLocal"`
}

type MapCodeConflict struct {
	AssetConflict
	CityCode string `json:"cityCode"`
}

type DownloaderErrorType string

const (
	InstallErrorInvalidAssetType   DownloaderErrorType = "install_invalid_asset_type"
	InstallErrorInvalidConfig      DownloaderErrorType = "install_invalid_config"
	InstallErrorInvalidMapCode     DownloaderErrorType = "install_invalid_map_code"
	InstallErrorRegistryLookup     DownloaderErrorType = "install_registry_lookup_failed"
	InstallErrorVersionLookup      DownloaderErrorType = "install_version_lookup_failed"
	InstallErrorVersionNotFound    DownloaderErrorType = "install_version_not_found"
	InstallErrorDownloadFailed     DownloaderErrorType = "install_download_failed"
	InstallErrorChecksumFailed     DownloaderErrorType = "install_checksum_failed"
	InstallErrorExtractFailed      DownloaderErrorType = "install_extract_failed"
	InstallErrorInvalidManifest    DownloaderErrorType = "install_invalid_manifest"
	InstallErrorInvalidArchive     DownloaderErrorType = "install_invalid_archive"
	InstallErrorMapCodeConflict    DownloaderErrorType = "install_map_code_conflict"
	InstallErrorFilesystem         DownloaderErrorType = "install_filesystem_error"
	InstallErrorPersistStateFailed DownloaderErrorType = "install_persist_state_failed"
	UninstallErrorInvalidAssetType DownloaderErrorType = "uninstall_invalid_asset_type"
	UninstallErrorNotInstalled     DownloaderErrorType = "uninstall_not_installed"
	UninstallErrorFilesystem       DownloaderErrorType = "uninstall_filesystem_error"
	UninstallErrorPersistState     DownloaderErrorType = "uninstall_persist_state_failed"
)

// List of deterministic install errors that should trigger automatic purge of the subscription without user confirmation, as they indicate the subscription is invalid/corrupt and cannot be resolved through retries.
var autoPurgeDownloadErrorTypes = map[DownloaderErrorType]struct{}{
	InstallErrorInvalidManifest: {},
	InstallErrorInvalidArchive:  {},
	InstallErrorChecksumFailed:  {},
	// TODO: Add another error if the map/mod salt is not present in the installed folder
}

func AutoPurgeDownloadErrors(err DownloaderErrorType) bool {
	_, ok := autoPurgeDownloadErrorTypes[err]
	return ok
}

type AssetInstallResponse struct {
	GenericResponse
	AssetType       AssetType           `json:"assetType"`
	AssetID         string              `json:"assetId"`
	Version         string              `json:"version"`
	Config          ConfigData          `json:"config,omitempty"`
	ErrorType       DownloaderErrorType `json:"errorType,omitempty"`
	MapCodeConflict *MapCodeConflict    `json:"mapCodeConflict,omitempty"`
}

type MapInstallOptions struct {
	ReplaceOnConflict bool `json:"replaceOnConflict"`
}

// Reserved for future mod-specific install options.
type ModInstallOptions struct{}

type InstallAssetRequest struct {
	AssetType AssetType          `json:"assetType"`
	AssetID   string             `json:"assetId"`
	Version   string             `json:"version"`
	Map       *MapInstallOptions `json:"map,omitempty"`
	Mod       *ModInstallOptions `json:"mod,omitempty"`
}

type AssetUninstallResponse struct {
	GenericResponse
	AssetType AssetType           `json:"assetType"`
	AssetID   string              `json:"assetId"`
	ErrorType DownloaderErrorType `json:"errorType,omitempty"`
}

// errorResponse is a helper to create a consistent error response
func ErrorResponse(msg string) GenericResponse {
	return GenericResponse{
		Status:  ResponseError,
		Message: msg,
	}
}

// successResponse is a helper to create a consistent success response
func SuccessResponse(msg string) GenericResponse {
	return GenericResponse{
		Status:  ResponseSuccess,
		Message: msg,
	}
}

// warnResponse is a helper to create a consistent warning response
func WarnResponse(msg string) GenericResponse {
	return GenericResponse{
		Status:  ResponseWarn,
		Message: msg,
	}
}

type AssetType string

const (
	AssetTypeMap AssetType = "map"
	AssetTypeMod AssetType = "mod"
)

var LocalMapCodePattern = regexp.MustCompile(`^[A-Z]{2,4}$`)

func IsValidAssetType(assetType AssetType) bool {
	switch assetType {
	case AssetTypeMap, AssetTypeMod:
		return true
	default:
		return false
	}
}

func AssetTypeDir(assetType AssetType) string {
	switch assetType {
	case AssetTypeMap:
		return "maps"
	case AssetTypeMod:
		return "mods"
	}
	panic("unsupported asset type: " + string(assetType))
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

func NormalizeSemver(version string) string {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "v") {
		return trimmed
	}
	return "v" + trimmed
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

// ProgressFunc is a callback for reporting download progress.
// itemId identifies what is being downloaded, received is bytes downloaded so far, total is the total size (-1 if unknown).
type ProgressFunc func(itemId string, received int64, total int64)

// progressReader wraps an io.Reader to report download progress via a callback.
type ProgressReader struct {
	Reader     io.Reader
	Total      int64
	Received   int64
	ItemId     string
	OnProgress ProgressFunc
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Received += int64(n)
	if pr.OnProgress != nil {
		pr.OnProgress(pr.ItemId, pr.Received, pr.Total)
	}
	return n, err
}
