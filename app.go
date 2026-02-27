package main

import (
	"context"
	"log"
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
	return &App{
		Registry:   registry,
		Config:     config,
		Downloader: NewDownloader(config, registry),
		Profiles: NewUserProfiles(),
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
	activeProfile := resolveStartupProfile(a)
	runStartupTasks(a, activeProfile)
}

func defaultStartupProfile() types.UserProfile {
	initial := types.InitialProfilesState()
	return initial.Profiles[initial.ActiveProfileID]
}

func resolveStartupProfile(a *App) types.UserProfile {
	// Ensure user profiles are initialized on startup.
	if err := a.Profiles.LoadProfiles(); err != nil {
		// TODO: Surface this error to the user in the UI and allow them to "reset" their profile state and bootstrap to defaults.
		log.Printf("Warning: failed to load user profiles, falling back to defaults for startup policy: %v", err)
	}

	activeProfile, err := a.Profiles.ResolveActiveProfile()
	if err != nil {
		log.Printf("Warning: failed to resolve active profile, falling back to defaults for startup policy: %v", err)
		return defaultStartupProfile()
	}
	return activeProfile
}

func runStartupTasks(a *App, activeProfile types.UserProfile) {
	if err := a.Registry.ensureLocalRepo(); err != nil {
		log.Printf("Warning: failed to ensure local registry repository: %v", err)
	}

	if activeProfile.SystemPreferences.RefreshRegistryOnStartup {
		if err := a.Registry.Refresh(); err != nil {
			log.Printf("Warning: failed to refresh registry on startup: %v", err)
		}
	}

	if activeProfile.SystemPreferences.AutoUpdateSubscriptions {
		if err := a.syncSubscriptions(activeProfile); err != nil {
			log.Printf("Warning: failed to sync subscriptions on startup: %v", err)
		}
	}
}

func (a *App) syncSubscriptions(profile types.UserProfile) error {
	log.Printf("TODO: implement startup subscription sync for profile %q", profile.ID)
	return nil
}

// shutdown is called when the app is shutting down
// We use this to save config and registry state to disk
func (a *App) shutdown(ctx context.Context) {
	a.Config.SaveConfig()
	a.Registry.WriteInstalledToDisk()
}
