package main

import (
	"railyard/internal/dialog"
	"railyard/internal/types"
)

func (a *App) OpenInFileExplorer(targetPath string) types.GenericResponse {
	return dialog.OpenInFileExplorer(targetPath)
}

func (a *App) OpenImportAssetDialog(assetType types.AssetType) types.ImportAssetDialogResponse {
	return dialog.OpenImportAssetDialog(a.ctx, assetType)
}
