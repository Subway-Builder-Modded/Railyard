package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"railyard/internal/types"
)

// App struct
type App struct {
	Registry   *Registry
	Config     *Config
	Downloader *Downloader
	ctx        context.Context
	Profiles *UserProfiles
	Logger   *AppLogger
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
		logger := NewAppLogger()
	return &App{
		Registry:   registry,
		Config:     config,
		Downloader: NewDownloader(config, registry),
		Profiles: NewUserProfiles(logger),
		Logger:   logger,
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

	if a.Logger == nil {
		a.Logger = NewAppLogger()
	}
	
	if err := moveLogFile(); err != nil {
		log.Printf("[WARN]: Failed to rotate startup log file: %v", err)
	}

	if err := a.Logger.Start(); err != nil {
		log.Printf("[WARN]: Failed to start app logger: %v", err)
	} else {
		a.Logger.SetRoutine(StartupRoutine)
	}

	runStartupRoutines(a)
	a.Logger.SetRoutine(RuntimeRoutine)
}

// shutdown is called when the app shuts down.
func (a *App) shutdown(ctx context.Context) {
	if a.Logger == nil {
		return
	}

	a.Logger.SetRoutine(ShutdownRoutine)
	a.Logger.logger().Info("application shutdown")

	if err := a.Logger.Shutdown(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to flush app logs on shutdown: %v\n", err)
	}
}

func resolveStartupProfile(a *App) types.UserProfile {
	if p, err := a.Profiles.LoadProfiles(); err == nil {
		return p
	} else {
		return a.recoverProfiles(err)
	}
}

func (a *App) recoverProfiles(cause error) types.UserProfile {
	success, quarantinedPath := a.Profiles.quarantineUserProfiles()
	if !success {
		a.Logger.Error("Failed to quarantine user profiles", cause)
		return types.DefaultProfile()
	}

	if resetErr := a.Profiles.ResetUserProfiles(); resetErr != nil {
		a.Logger.Error("Failed to reset user profiles", resetErr, "cause", cause, "quarantinedPath", quarantinedPath)
		return types.DefaultProfile()
	}

	profile, resolveErr := a.Profiles.ResolveActiveProfile()
	if resolveErr != nil {
		a.Logger.Error("Failed to resolve active profile after reset", resolveErr, "cause", cause)
		return types.DefaultProfile()
	}

	a.Logger.Warn("Recovered user profiles using defaults after load failure", "quarantinedPath", quarantinedPath)
	return profile
}

func runStartupRoutines(a *App) {
	// TODO: Handle auto-update of application version...

	activeProfile := resolveStartupProfile(a)

	// TODO: Backend should control registry state; frontend should not force initialization of the registry on startup.
	if err := a.Registry.initialize(); err != nil {
		a.Logger.Warn("Failed to ensure local registry repository", "error", err)
	}

	if activeProfile.SystemPreferences.RefreshRegistryOnStartup {
		if err := a.Registry.Refresh(); err != nil {
			a.Logger.Warn("Failed to refresh registry on startup", "error", err)
		}
	}

	if activeProfile.SystemPreferences.AutoUpdateSubscriptions {
		if err := a.syncSubscriptions(activeProfile); err != nil {
			a.Logger.Warn("Failed to sync subscriptions on startup", "error", err)
		}
	}
}

func (a *App) syncSubscriptions(profile types.UserProfile) error {
	a.Logger.Info("TODO: implement startup subscription sync", "profile", profile.ID)
	// Pseudocode
	// installedMods, installedMaps := a.Registry.GetInstalledMods(), a.Registry.GetInstalledMaps()
	// for map in mapsToUpdate => HandleInstall(id, version, "map")
	// for mod in modsToUpdate => HandleInstall(id, version, "mod")
	// for map in mapsToDelete => HandleUninstall(id, "map")
	// for map in modsToDelete => HandleUninstall(id, "mod")
	// Compile errors from all operations and return a joined error
	return nil
}

// shutdown is called when the app is shutting down
// We use this to save config and registry state to disk
func (a *App) shutdown(ctx context.Context) {
	a.Config.SaveConfig()
	a.Registry.WriteInstalledToDisk()
}
