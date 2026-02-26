package main

import (
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"slices"
	"strings"
	"time"
)

// App struct
type App struct {
	ctx      context.Context
	Registry *Registry
}

type MissingFilesError struct {
	Files []string
}

func (e *MissingFilesError) Error() string {
	return "Missing required files: " + strings.Join(e.Files, ", ")
}

type MapAlreadyExistsError struct {
	MapCode string
}

func (e *MapAlreadyExistsError) Error() string {
	return "Map with code '" + e.MapCode + "' has already been installed or would overwrite a vanilla map."
}

type FileFoundStruct struct {
	found      bool
	fileObject *zip.File
	required   bool
}

type ConfigData struct {
	Name             string      `json:"name"`
	Code             string      `json:"code"`
	Description      string      `json:"description"`
	Population       int         `json:"population"`
	Country          *string     `json:"country"`
	ThumbnailBbox    *[4]float64 `json:"thumbnail_bbox"`
	Creator          string      `json:"creator"`
	Version          string      `json:"version"`
	InitialViewState struct {
		Latitude  float64  `json:"latitude"`
		Longitude float64  `json:"longitude"`
		Zoom      float64  `json:"zoom"`
		Pitch     *float64 `json:"pitch"`
		Bearing   float64  `json:"bearing"`
	} `json:"initial_view_state"`
}

type InstallMapResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    *ConfigData `json:"data,omitempty"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		Registry: NewRegistry(),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize the registry (clone or update) on startup
	if err := a.Registry.Initialize(); err != nil {
		log.Printf("Warning: failed to initialize registry: %v", err)
	}
}

func (a *App) InstallMap(zipFilePath string, subwayBuilderDataPath string) InstallMapResponse {
	reader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return InstallMapResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to open zip file: %v", err),
		}
	}
	defer reader.Close()

	filesFound := map[string]FileFoundStruct{
		"config":     {found: false, fileObject: nil, required: true},
		"demandData": {found: false, fileObject: nil, required: true},
		"roads":      {found: false, fileObject: nil, required: true},
		"runways":    {found: false, fileObject: nil, required: true},
		"buildings":  {found: false, fileObject: nil, required: true},
		"tiles":      {found: false, fileObject: nil, required: true},
		"oceanDepth": {found: false, fileObject: nil, required: false},
		"thumbnail":  {found: false, fileObject: nil, required: false},
	}

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		fileFound := ""
		switch file.Name {
		case "config.json":
			fileFound = "config"
		case "demand_data.json":
			fileFound = "demandData"
		case "roads.geojson":
			fileFound = "roads"
		case "runways_taxiways.geojson":
			fileFound = "runways"
		case "buildings_index.json":
			fileFound = "buildings"
		case "ocean_depth_index.json":
			fileFound = "oceanDepth"
		}
		if strings.HasSuffix(file.Name, ".pmtiles") {
			fileFound = "tiles"
		}
		if strings.HasSuffix(file.Name, ".svg") {
			fileFound = "thumbnail"
		}
		if fileFound != "" {
			filesFound[fileFound] = FileFoundStruct{found: true, fileObject: file, required: filesFound[fileFound].required}
		}
	}

	missingRequiredFiles := []string{}
	for key, fileInfo := range filesFound {
		if fileInfo.required && !fileInfo.found {
			missingRequiredFiles = append(missingRequiredFiles, key)
		}
	}
	if len(missingRequiredFiles) > 0 {
		return InstallMapResponse{
			Status:  "error",
			Message: "Missing required files: " + strings.Join(missingRequiredFiles, ", "),
		}
	}

	configFile, err := filesFound["config"].fileObject.Open()
	if err != nil {
		return InstallMapResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to open config file: %v", err),
		}
	}
	defer configFile.Close()

	fileBytes, err := io.ReadAll(configFile)
	if err != nil {
		return InstallMapResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to read config file: %v", err),
		}
	}

	var configData ConfigData
	err = json.Unmarshal(fileBytes, &configData)
	if err != nil {
		return InstallMapResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to parse config file: %v", err),
		}
	}

	installedMaps := a.Registry.GetInstalledMapCodes()
	vanillaMaps := a.GetVanillaMapCodes()

	if slices.Contains(installedMaps, configData.Code) || slices.Contains(vanillaMaps, configData.Code) {
		return InstallMapResponse{
			Status:  "error",
			Message: "Map with code '" + configData.Code + "' has already been installed or would overwrite a vanilla map.",
		}
	}

	os.MkdirAll(path.Join(subwayBuilderDataPath, "cities", "data", configData.Code), os.ModePerm)

	// Channel to collect errors from all goroutines
	errorChan := make(chan error, len(filesFound))
	var activeGoroutines int

	// Process each file (except config) in its own goroutine for maximum parallelization
	for entry, fileInfo := range filesFound {
		if fileInfo.found && entry != "config" {
			activeGoroutines++
			go func(entry string, fileInfo FileFoundStruct) {
				defer func() {
					// Always send to channel to signal completion (nil for success)
					if r := recover(); r != nil {
						errorChan <- fmt.Errorf("Panic in %s processing: %v", entry, r)
					}
				}()

				log.Printf("[DEBUG] Starting %s goroutine...", entry)
				srcFile, err := fileInfo.fileObject.Open()
				if err != nil {
					log.Printf("[ERROR] Failed to open %s file: %v", entry, err)
					errorChan <- fmt.Errorf("Failed to open file %s: %v", entry, err)
					return
				}
				defer srcFile.Close()
				log.Printf("[DEBUG] Successfully opened %s file", entry)

				// Handle different file types
				switch entry {
				case "tiles":
					userConfigDir, err := os.UserConfigDir()
					if err != nil {
						errorChan <- fmt.Errorf("Failed to get user config directory for tiles: %v", err)
						return
					}

					tilesDir := path.Join(userConfigDir, "railyard", "tiles")
					err = os.MkdirAll(tilesDir, os.ModePerm)
					if err != nil {
						errorChan <- fmt.Errorf("Failed to create tiles directory: %v", err)
						return
					}

					destFilePath := path.Join(tilesDir, configData.Code+".pmtiles")
					log.Printf("Installing %s for map %s at %s", entry, configData.Code, destFilePath)
					destFile, err := os.Create(destFilePath)
					if err != nil {
						errorChan <- fmt.Errorf("Failed to create destination file for tiles: %v", err)
						return
					}
					defer destFile.Close()

					_, err = io.Copy(destFile, srcFile)
					if err != nil {
						errorChan <- fmt.Errorf("Failed to copy tiles file: %v", err)
						return
					}
					log.Printf("Successfully installed %s for map %s", entry, configData.Code)

				case "thumbnail":
					cityMapsExists, err := os.Stat(path.Join(subwayBuilderDataPath, "public", "data", "city-maps"))
					if os.IsNotExist(err) || !cityMapsExists.IsDir() {
						err = os.MkdirAll(path.Join(subwayBuilderDataPath, "public", "data", "city-maps"), os.ModePerm)
						if err != nil {
							errorChan <- fmt.Errorf("Failed to create city-maps directory: %v", err)
							return
						}
					}
					destFilePath := path.Join(subwayBuilderDataPath, "public", "data", "city-maps", configData.Code+".svg")
					log.Printf("Installing %s for map %s at %s", entry, configData.Code, destFilePath)
					destFile, err := os.Create(destFilePath)
					if err != nil {
						errorChan <- fmt.Errorf("Failed to create destination file for thumbnail: %v", err)
						return
					}
					defer destFile.Close()

					_, err = io.Copy(destFile, srcFile)
					if err != nil {
						errorChan <- fmt.Errorf("Failed to copy thumbnail file: %v", err)
						return
					}
					log.Printf("Successfully installed %s for map %s", entry, configData.Code)

				default:
					// Handle compressed files (demandData, roads, runways, buildings, oceanDepth)
					destFilePath := path.Join(subwayBuilderDataPath, "cities", "data", configData.Code, path.Base(fileInfo.fileObject.Name)+".gz")
					fileSize := fileInfo.fileObject.UncompressedSize64
					log.Printf("Installing %s for map %s at %s (size: %.2f MB)", entry, configData.Code, destFilePath, float64(fileSize)/(1024*1024))

					destFile, err := os.Create(destFilePath)
					if err != nil {
						errorChan <- fmt.Errorf("Failed to create destination file for %s: %v", entry, err)
						return
					}
					defer destFile.Close()

					// Use fastest compression level for better performance
					compressedWriter, err := gzip.NewWriterLevel(destFile, gzip.BestSpeed)
					if err != nil {
						errorChan <- fmt.Errorf("Failed to create gzip writer for %s: %v", entry, err)
						return
					}
					defer compressedWriter.Close()

					log.Printf("[DEBUG] Starting compression for %s (%.2f MB)...", entry, float64(fileSize)/(1024*1024))
					startTime := time.Now()

					_, err = io.Copy(compressedWriter, srcFile)
					if err != nil {
						errorChan <- fmt.Errorf("Failed to copy and compress file %s: %v", entry, err)
						return
					}

					duration := time.Since(startTime)
					log.Printf("Successfully installed %s for map %s (compressed in %v)", entry, configData.Code, duration)
				}

				// Signal successful completion
				errorChan <- nil
			}(entry, fileInfo)
		}
	}

	// Wait for all goroutines to complete
	log.Printf("Waiting for %d file processing goroutines to complete...", activeGoroutines)
	for i := 0; i < activeGoroutines; i++ {
		select {
		case err := <-errorChan:
			if err != nil {
				log.Printf("[ERROR] File processing failed: %v", err)
				return InstallMapResponse{
					Status:  "error",
					Message: err.Error(),
				}
			}
			log.Printf("[DEBUG] File processing goroutine %d/%d completed successfully", i+1, activeGoroutines)
		case <-time.After(10 * time.Minute):
			log.Printf("[ERROR] File processing timed out after 10 minutes")
			return InstallMapResponse{
				Status:  "error",
				Message: "File processing timed out after 10 minutes",
			}
		}
	}

	log.Printf("[DEBUG] All file processing completed successfully")
	return InstallMapResponse{
		Status: "success",
		Data:   &configData,
	}
}

// GetVanillaMapCodes returns the city codes of maps included with the app.
// Currently stubbed to return an empty slice.
func (a *App) GetVanillaMapCodes() []string {
	return []string{}
}
