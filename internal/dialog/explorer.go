package dialog

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"railyard/internal/paths"
	"railyard/internal/types"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func OpenInFileExplorer(targetPath string) types.GenericResponse {
	trimmedPath := strings.TrimSpace(targetPath)
	if trimmedPath == "" {
		return types.ErrorResponse("invalid path")
	}
	cleanedPath := paths.NormalizeLocalPath(trimmedPath)
	if cleanedPath == "" {
		return types.ErrorResponse("invalid path")
	}

	info, err := os.Stat(cleanedPath)
	if err != nil {
		return types.ErrorResponse(fmt.Sprintf("failed to resolve path: %v", err))
	}
	if !info.IsDir() {
		cleanedPath = filepath.Dir(cleanedPath)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", cleanedPath)
	case "darwin":
		cmd = exec.Command("open", cleanedPath)
	default:
		cmd = exec.Command("xdg-open", cleanedPath)
	}

	if err := cmd.Start(); err != nil {
		return types.ErrorResponse(fmt.Sprintf("failed to open path in file explorer: %v", err))
	}
	return types.SuccessResponse("opened in file explorer")
}

func OpenImportAssetArchive(ctx context.Context) (string, error) {
	return wruntime.OpenFileDialog(ctx, wruntime.OpenDialogOptions{
		Title: "Import Map Archive",
		Filters: []wruntime.FileFilter{
			{
				DisplayName: "ZIP Archives",
				Pattern:     "*.zip",
			},
		},
	})
}

func OpenImportAssetDialog(ctx context.Context, assetType types.AssetType) types.ImportAssetDialogResponse {
	selectedPath, err := OpenImportAssetArchive(ctx)
	if err != nil {
		return types.ImportAssetDialogResponse{
			GenericResponse: types.ErrorResponse(fmt.Sprintf("Failed to open import dialog: %v", err)),
			Path:            "",
		}
	}
	if selectedPath == "" {
		return types.ImportAssetDialogResponse{
			GenericResponse: types.WarnResponse("Import cancelled"),
			Path:            "",
		}
	}
	return types.ImportAssetDialogResponse{
		GenericResponse: types.SuccessResponse("Asset archive selected"),
		Path:            selectedPath,
	}
}
