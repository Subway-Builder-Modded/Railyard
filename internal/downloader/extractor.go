package downloader

import (
	"archive/zip"
	"io"
	"os"
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"

	"railyard/internal/constants"
	"railyard/internal/files"
	"railyard/internal/paths"
	"railyard/internal/types"
	"railyard/internal/utils"
)

type mapArtifactTarget struct {
	Key            string
	DestinationDir func(*Downloader) string
	Suffix         string
}

var mapArtifactTargets = []mapArtifactTarget{
	{
		Key:            files.MapArchiveKeyTiles,
		DestinationDir: (*Downloader).getMapTilePath,
		Suffix:         files.MapTileFileExt,
	},
	{
		Key:            files.MapArchiveKeyThumbnail,
		DestinationDir: (*Downloader).getMapThumbnailPath,
		Suffix:         files.MapThumbnailFileExt,
	},
}

func reportExtractProgress(fn ExtractProgressFunc, itemID string, extracted int64, total int64) {
	if fn == nil {
		return
	}
	fn(itemID, extracted, total)
}

// extractMod processes the downloaded mod zip file, extracts it to the appropriate location.
func extractMod(d *Downloader, filePath string, modId string, version string) types.AssetInstallResponse {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorInvalidArchive, "Failed to open zip file", err, "file_path", filePath, "mod_id", modId)
	}
	defer reader.Close()

	destFolder := paths.JoinLocalPath(d.getModPath(), modId)

	requiredFiles := map[string]types.FileFoundStruct{
		"manifest":        {Found: false, FileObject: nil, Required: true},
		"manifest_target": {Found: false, FileObject: nil, Required: true},
	}

	fileCount := 0
	for _, file := range reader.File {
		if !file.FileInfo().IsDir() {
			fileCount++
		}
		if file.Name == constants.MANIFEST_JSON {
			requiredFiles["manifest"] = types.FileFoundStruct{Found: true, FileObject: file, Required: true}
		}
	}

	if !requiredFiles["manifest"].Found {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorInvalidArchive, "Zip file is missing manifest.json", nil, "file_path", filePath, "mod_id", modId)
	}

	rawManifestReader, err := requiredFiles["manifest"].FileObject.Open()
	if err != nil {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorInvalidManifest, "Failed to read manifest file", err, "file_path", filePath, "mod_id", modId)
	}
	defer rawManifestReader.Close()

	rawManifestBytes, err := io.ReadAll(rawManifestReader)
	if err != nil {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorInvalidManifest, "Failed to read manifest file", err, "file_path", filePath, "mod_id", modId)
	}

	manifestData, err := files.ParseJSON[types.MetroMakerModManifest](rawManifestBytes, constants.MANIFEST_JSON)
	if err != nil {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorInvalidManifest, "Failed to parse manifest file", err, "file_path", filePath, "mod_id", modId)
	}
	for _, file := range reader.File {
		if file.Name == manifestData.Main {
			requiredFiles["manifest_target"] = types.FileFoundStruct{Found: true, FileObject: file, Required: true}
			break
		}
	}

	if !requiredFilesPresent(requiredFiles) {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorInvalidArchive, "Zip file is missing one or more required files", nil, "file_path", filePath, "mod_id", modId)
	}

	reportExtractProgress(d.OnExtractProgress, modId, 0, int64(fileCount))
	var installCounter atomic.Int64
	if err := files.WritePathsAtomically([]files.AtomicWrite{files.AtomicDirectoryWrite{
		Path:  destFolder,
		Label: "mod install directory",
		Callback: func(stagingFolder string) error {
			// First pass: create directories to avoid extract errors
			for _, file := range reader.File {
				if !file.FileInfo().IsDir() {
					continue
				}
				destPath := paths.JoinLocalPath(stagingFolder, file.Name)
				if err := os.MkdirAll(destPath, os.ModePerm); err != nil {
					return err
				}
			}

			// Second pass: extract files in parallel
			var wg sync.WaitGroup
			errChan := make(chan error, len(reader.File))
			for _, file := range reader.File {
				if file.FileInfo().IsDir() {
					continue
				}
				wg.Add(1)
				go func(file *zip.File) {
					defer wg.Done()

					destPath := paths.JoinLocalPath(stagingFolder, file.Name)
					parentDir := filepath.Dir(destPath)
					if err := os.MkdirAll(parentDir, os.ModePerm); err != nil {
						errChan <- err
						return
					}

					destFile, err := os.Create(destPath)
					if err != nil {
						errChan <- err
						return
					}
					defer destFile.Close()

					srcFile, err := file.Open()
					if err != nil {
						errChan <- err
						return
					}
					defer srcFile.Close()

					_, err = io.Copy(destFile, srcFile)
					reportExtractProgress(d.OnExtractProgress, modId, installCounter.Add(1), int64(fileCount))
					if err != nil {
						errChan <- err
					}
				}(file)
			}

			wg.Wait()
			close(errChan)
			if len(errChan) > 0 {
				return <-errChan
			}

			return createAssetMarker(paths.JoinLocalPath(stagingFolder, constants.RailyardAssetMarker))
		},
	}}); err != nil {
		return d.installError(types.AssetTypeMod, modId, version, types.ConfigData{}, types.InstallErrorExtractFailed, "Failed to extract mod zip", err, "file_path", filePath, "mod_id", modId)
	}

	return d.installSuccess(types.AssetTypeMod, modId, version, types.ConfigData{}, "Mod extracted successfully", "file_path", filePath, "assetId", modId)
}

// extractMap processes map zip files for downloaded/local installs and writes only the expected city-data artifacts.
func extractMap(d *Downloader, filePath string, mapId string, version string) types.AssetInstallResponse {
	configData, errorType, inspectErr := files.ValidateMapArchive(filePath)
	if inspectErr != nil {
		return d.installError(types.AssetTypeMap, mapId, version, configData, errorType, "Failed map archive inspection", inspectErr, "file_path", filePath)
	}

	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return d.installError(types.AssetTypeMap, mapId, version, configData, types.InstallErrorInvalidArchive, "Failed to open zip file", err, "file_path", filePath)
	}
	defer reader.Close()

	filesFound := files.BuildMapArchiveFileIndex(reader.File)

	filesCount := 0
	for _, fileStruct := range filesFound {
		if fileStruct.Found {
			filesCount++
		}
	}
	if configData.ThumbnailBbox != nil {
		if fileStruct, ok := filesFound[files.MapArchiveKeyThumbnail]; !ok || !fileStruct.Found {
			filesCount++
		}
	}

	if conflict, hasConflict := d.FindMapCodeConflict(mapId, configData.Code, true); hasConflict {
		return d.installError(
			types.AssetTypeMap,
			mapId,
			version,
			configData,
			types.InstallErrorMapCodeConflict,
			"Cannot install map because its code matches a vanilla map included with the game or an already installed map.",
			nil,
			"map_code", conflict.CityCode,
			"conflicting_asset_id", conflict.ExistingAssetID,
			"conflicting_is_local", conflict.ExistingIsLocal,
		)
	}

	// Create necessary directories first
	destFolder := paths.JoinLocalPath(d.getMapDataPath(), configData.Code)
	if err := os.MkdirAll(d.getMapTilePath(), os.ModePerm); err != nil {
		return d.installError(types.AssetTypeMap, mapId, version, configData, types.InstallErrorFilesystem, "Failed to create tiles directory", err, "tiles_path", d.getMapTilePath())
	}
	if err := os.MkdirAll(d.getMapThumbnailPath(), os.ModePerm); err != nil {
		return d.installError(types.AssetTypeMap, mapId, version, configData, types.InstallErrorFilesystem, "Failed to create thumbnail directory", err, "thumbnail_path", d.getMapThumbnailPath())
	}

	// Use atomic counter to track progress across routines
	var extractCount atomic.Int64
	reportExtractProgress(d.OnExtractProgress, configData.Code, 0, int64(filesCount))
	// Determine set of atomic writes needed to be made for the main map directory
	atomicWrites := []files.AtomicWrite{
		files.AtomicDirectoryWrite{
			Path:  destFolder,
			Label: "map data directory",
			Callback: func(stagingFolder string) error {
				var wg sync.WaitGroup
				errChan := make(chan error, len(filesFound))

				for key, fileStruct := range filesFound {
					// Skip tile/thumbnail artifacts in the main loop since they are saved in a separate Railyard-managed directory.
					if !fileStruct.Found || key == files.MapArchiveKeyTiles || key == files.MapArchiveKeyThumbnail {
						continue
					}

					wg.Add(1)
					go func(key string, fileStruct types.FileFoundStruct) {
						defer wg.Done()

						srcFile, err := fileStruct.FileObject.Open()
						if err != nil {
							errChan <- err
							return
						}
						defer srcFile.Close()

						outputFileName := path.Base(fileStruct.FileObject.Name)
						destinationPath := paths.JoinLocalPath(stagingFolder, outputFileName+".gz")
						shouldArchive := true
						// Extract out config.json for future bootstrapping from installed state, in particular for local maps.
						if key == files.MapArchiveKeyConfig {
							destinationPath = paths.JoinLocalPath(stagingFolder, files.MapConfigFileName)
							shouldArchive = false
						}

						if err := files.WriteArchiveStream(destinationPath, srcFile, shouldArchive); err != nil {
							errChan <- err
							return
						}
						reportExtractProgress(d.OnExtractProgress, configData.Code, extractCount.Add(1), int64(filesCount))
					}(key, fileStruct)
				}

				wg.Wait()
				close(errChan)
				if len(errChan) > 0 {
					return <-errChan
				}

				return createAssetMarker(paths.JoinLocalPath(stagingFolder, constants.RailyardAssetMarker))
			},
		},
	}

	stagedTempPaths := make([]string, 0, 2)
	defer func() {
		for _, tempPath := range stagedTempPaths {
			_ = os.Remove(tempPath)
		}
	}()

	// Append atomic writes for file and thumbnail paths
	for _, stagedFile := range mapArtifactTargets {
		fileStruct, ok := filesFound[stagedFile.Key]
		if !ok || !fileStruct.Found {
			continue
		}

		destinationPath := paths.JoinLocalPath(stagedFile.DestinationDir(d), configData.Code+stagedFile.Suffix)
		stagedPath, err := files.StageArchiveForAtomicWrite(destinationPath, fileStruct.FileObject, false)
		if err != nil {
			return d.installError(types.AssetTypeMap, mapId, version, configData, types.InstallErrorExtractFailed, "Failed to stage map artifact", err, "file_path", filePath, "artifact_key", stagedFile.Key)
		}
		stagedTempPaths = append(stagedTempPaths, stagedPath)
		atomicWrites = append(atomicWrites, files.AtomicFileWrite{
			Path:       destinationPath,
			Label:      "map artifact",
			StagedPath: stagedPath,
		})
		reportExtractProgress(d.OnExtractProgress, configData.Code, extractCount.Add(1), int64(filesCount))
	}

	// Perform atomic writes to move all staged files to the final location.
	if err := files.WritePathsAtomically(atomicWrites); err != nil {
		return d.installError(types.AssetTypeMap, mapId, version, configData, types.InstallErrorExtractFailed, "Failed to atomically install map artifacts", err, "file_path", filePath)
	}

	if fileStruct, ok := filesFound[files.MapArchiveKeyThumbnail]; (!ok || !fileStruct.Found) && configData.ThumbnailBbox != nil {
		srv, port, srvErr := utils.StartTempPMTilesServer()
		if srvErr != nil {
			return d.installWarn(types.AssetTypeMap, mapId, version, configData, nil, "Failed to start PMTiles server for thumbnail generation, but map was extracted successfully.", "file_path", filePath, "map_code", configData.Code)
		}
		defer srv.Close()

		thumbnailData, err := utils.GenerateThumbnail(configData.Code, configData, port)
		if err != nil {
			return d.installWarn(types.AssetTypeMap, mapId, version, configData, nil, "Failed to generate thumbnail, but map was extracted successfully. You can try generating the thumbnail later from the map details page.", "file_path", filePath, "map_code", configData.Code)
		}

		thumbnailPath := paths.JoinLocalPath(d.getMapThumbnailPath(), configData.Code+files.MapThumbnailFileExt)
		if err := files.WriteFilesAtomically([]files.AtomicFileWrite{
			{
				Path:  thumbnailPath,
				Label: "map thumbnail",
				Data:  []byte(thumbnailData),
				Perm:  0o644,
			},
		}); err != nil {
			return d.installWarn(types.AssetTypeMap, mapId, version, configData, nil, "Failed to save generated thumbnail, but map was extracted successfully. You can try generating the thumbnail later from the map details page.", "file_path", filePath, "map_code", configData.Code, "thumbnail_path", thumbnailPath)
		}
		reportExtractProgress(d.OnExtractProgress, configData.Code, extractCount.Add(1), int64(filesCount))
	}

	return d.installSuccess(types.AssetTypeMap, mapId, version, configData, "Map extracted successfully", "file_path", filePath, "map_code", configData.Code)
}

func createAssetMarker(path string) error {
	return os.WriteFile(path, nil, 0o644)
}
