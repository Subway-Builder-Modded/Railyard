package main

import (
	"context"
	"log"
	"strings"

	"railyard/internal/types"
)

type MissingFilesError struct {
	Files []string
}

// App struct
type App struct {
	Registry   *Registry
	Config     *Config
	Downloader *Downloader
	ctx        context.Context
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

type installMapResponse struct {
	Status  string            `json:"status"`
	Message string            `json:"message,omitempty"`
	Data    *types.ConfigData `json:"data,omitempty"`
}

type installModResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type HandleInstallResponse struct {
	Status  string            `json:"status"`
	Message string            `json:"message,omitempty"`
	Data    *types.ConfigData `json:"data,omitempty"`
}

// CityInfo represents information about a single city

// NewApp creates a new App application struct
func NewApp() *App {
	config := NewConfig()
	registry := NewRegistry()
	return &App{
		Registry:   registry,
		Config:     config,
		Downloader: NewDownloader(config, registry),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.Config.setContext(ctx)
	if _, err := a.Config.resolveConfig(); err != nil {
		log.Printf("Warning: failed to resolve config on startup: %v", err)
	}

	// Initialize the registry (clone or update) on startup
	if err := a.Registry.Initialize(); err != nil {
		log.Printf("Warning: failed to initialize registry: %v", err)
	}
}

func (a *App) shutdown(ctx context.Context) {
	a.Config.SaveConfig()
	a.Registry.WriteInstalledToDisk()
}
