package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"

	"railyard/internal/config"
	"railyard/internal/constants"
	"railyard/internal/downloader"
	"railyard/internal/logger"
	"railyard/internal/paths"
	"railyard/internal/profiles"
	"railyard/internal/registry"
	"railyard/internal/types"
	"railyard/internal/updater"
	"railyard/internal/utils"

	"github.com/protomaps/go-pmtiles/pmtiles"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	Registry   *registry.Registry
	Config     *config.Config
	Downloader *downloader.Downloader
	ctx        context.Context
	Profiles   *profiles.UserProfiles
	Logger     *logger.AppLogger

	gameMu        sync.Mutex
	gameCmd       *exec.Cmd
	pmtilesServer *http.Server
	startupMu     sync.RWMutex
	startupReady  bool
}

// NewApp creates a new App application struct
func NewApp() *App {
	cfg := config.NewConfig()
	l := logger.NewAppLogger()
	reg := registry.NewRegistry(l)
	dl := downloader.NewDownloader(cfg, reg, l)
	return &App{
		Registry:   reg,
		Config:     cfg,
		Downloader: dl,
		Profiles:   profiles.NewUserProfiles(reg, dl, l, cfg),
		Logger:     l,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.setStartupReady(false)
	a.ctx = ctx
	a.Downloader.SetContext(ctx)
	a.Config.SetContext(ctx)
	a.Downloader.OnExtractProgress = func(itemId string, extracted int64, total int64) {
		wailsruntime.EventsEmit(ctx, "extract:progress", map[string]interface{}{
			"itemId":          itemId,
			"amountExtracted": extracted,
			"total":           total,
		})
	}
	a.Downloader.OnProgress = func(itemId string, received int64, total int64) {
		wailsruntime.EventsEmit(ctx, "download:progress", map[string]interface{}{
			"itemId":   itemId,
			"received": received,
			"total":    total,
		})
	}
	if _, err := a.Config.ResolveConfig(); err != nil {
		log.Printf("Warning: failed to resolve config on startup: %v", err)
	}

	if a.Logger == nil {
		a.Logger = logger.NewAppLogger()
	}

	if err := paths.MoveLogFile(); err != nil {
		log.Printf("[WARN]: Failed to rotate startup log file: %v", err)
	}

	if err := a.Logger.Start(); err != nil {
		log.Printf("[WARN]: Failed to start app logger: %v", err)
	}

	activeProfile := resolveStartupProfile(a)
	a.Logger.Info("Active user profile loaded on startup", "profile_id", activeProfile.ID)

	if err := a.Registry.Initialize(); err != nil {
		a.Logger.Warn("Failed to ensure local registry repository", "error", err)
	}

	if a.Config.Cfg.CheckForUpdatesOnLaunch {
		updater.CheckForUpdates(a.ctx, a.Downloader.OnProgress, a.Logger)
	}

	// Registry must be initialized + startup profile ready so that initial Frontend state is viable.
	a.setStartupReady(true)
	go runNonBlockingStartupRoutines(a, activeProfile)
}

func (a *App) setStartupReady(ready bool) {
	a.startupMu.Lock()
	defer a.startupMu.Unlock()
	a.startupReady = ready
}

// IsStartupReady reports whether backend startup routines have completed.
func (a *App) IsStartupReady() bool {
	a.startupMu.RLock()
	defer a.startupMu.RUnlock()
	return a.startupReady
}

// shutdown is called when the app shuts down.
func (a *App) shutdown(ctx context.Context) {
	if a.Logger == nil {
		return
	}

	a.Logger.Info("application shutdown")

	if err := a.Logger.Shutdown(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to flush app logs on shutdown: %v\n", err)
	}

	if _, err := a.Config.SaveConfig(); err != nil {
		log.Printf("Warning: failed to save config on shutdown: %v", err)
	}
	if err := a.Registry.WriteInstalledToDisk(); err != nil {
		log.Printf("Warning: failed to persist installed registry state on shutdown: %v", err)
	}

}

func resolveStartupProfile(a *App) types.UserProfile {
	loadResult := a.Profiles.LoadProfiles()
	if loadResult.Status == types.ResponseSuccess {
		return loadResult.Profile
	}
	return a.recoverProfiles(loadResult)
}

func (a *App) recoverProfiles(cause types.UserProfileResult) types.UserProfile {
	success, quarantinedPath := a.Profiles.QuarantineUserProfiles()
	if !success {
		a.Logger.MultipleError("Failed to quarantine user profiles", logger.AsErrors(cause.Errors), "cause", cause.Message, "quarantinedPath", quarantinedPath)
		return types.DefaultProfile()
	}

	resetResult := a.Profiles.ResetUserProfiles()
	if resetResult.Status == types.ResponseError {
		a.Logger.MultipleError("Failed to reset user profiles", logger.AsErrors(resetResult.Errors), "cause", cause.Message, "quarantinedPath", quarantinedPath)
		return types.DefaultProfile()
	}

	a.Logger.Warn("Recovered user profiles using defaults after load failure", "quarantinedPath", quarantinedPath)
	if resetResult.Profile.ID == "" {
		return types.DefaultProfile()
	}
	return resetResult.Profile
}

func runNonBlockingStartupRoutines(a *App, activeProfile types.UserProfile) {
	wailsruntime.WindowMaximise(a.ctx)
	if activeProfile.SystemPreferences.RefreshRegistryOnStartup {
		if err := a.Registry.Refresh(); err != nil {
			a.Logger.Warn("Failed to refresh registry on startup", "error", err)
		}
	}

	// Sync subscriptions for active profile on startup
	// TODO: Make this configurable within the profile itself
	syncResult := a.Profiles.SyncSubscriptions(activeProfile.ID)
	switch syncResult.Status {
	case types.ResponseError:
		a.Logger.MultipleError("Failed to sync profile subscriptions on startup", logger.AsErrors(syncResult.Errors), "profile_id", activeProfile.ID)
	case types.ResponseWarn:
		a.Logger.Warn("Profile subscriptions synced with warnings on startup", "message", syncResult.Message, "profile_id", activeProfile.ID, "error_count", len(syncResult.Errors))
	}
}

// GetGameVersion attempts to detect the installed Subway Builder version.
// Returns empty string if detection fails.
func (a *App) GetGameVersion() string {
	cfg := a.Config.GetConfig()
	if !cfg.Validation.ExecutablePathValid {
		return ""
	}
	exePath := cfg.Config.ExecutablePath

	var candidates []string
	if runtime.GOOS == "darwin" {
		// macOS: exe is at <app>/Contents/MacOS/<name>, resources at <app>/Contents/Resources/app/package.json
		macosDir := path.Dir(exePath)
		contentsDir := path.Dir(macosDir)
		candidates = append(candidates,
			path.Join(contentsDir, "Resources", "app", "package.json"),
		)
	} else {
		// Windows/Linux: exe is alongside resources/ directory
		exeDir := path.Dir(exePath)
		candidates = append(candidates,
			path.Join(exeDir, "resources", "app", "package.json"),
		)
	}

	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		var pkg struct {
			Version string `json:"version"`
		}
		if err := json.Unmarshal(data, &pkg); err != nil {
			continue
		}
		if pkg.Version != "" {
			return pkg.Version
		}
	}
	return ""
}

func (a *App) LaunchGame() error {
	a.gameMu.Lock()
	if a.gameCmd != nil && a.gameCmd.ProcessState == nil {
		a.gameMu.Unlock()
		return fmt.Errorf("game is already running")
	}
	a.gameMu.Unlock()

	cfg := a.Config.GetConfig()
	if !cfg.Validation.ExecutablePathValid {
		return fmt.Errorf("game executable path is not configured or invalid")
	}

	port, err := a.startPMTilesServer()
	if err != nil {
		a.Logger.Warn("Failed to start PMTiles server", "error", err)
		return err
	}

	wailsruntime.EventsEmit(a.ctx, "server:port", port)
	a.Logger.Info(fmt.Sprintf("Debug thumbnails: http://127.0.0.1:%d/debug/thumbnails", port))

	a.generateMissingThumbnails(port)

	if err := a.generateMod(port); err != nil {
		a.Logger.Warn("Failed to generate mod", "error", err)
		return err
	}

	exePath := cfg.Config.ExecutablePath
	a.Logger.Info("Launching game", "path", exePath)

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" && (strings.HasSuffix(exePath, ".app") || strings.Contains(exePath, ".app/")) {
		// On macOS, resolve .app bundle to the inner executable and launch via shell
		// to handle Electron stub executables that lack valid magic bytes
		innerExe := exePath
		if strings.HasSuffix(exePath, ".app") {
			// Derive inner binary from Info.plist CFBundleExecutable convention
			bundleName := strings.TrimSuffix(path.Base(exePath), ".app")
			innerExe = path.Join(exePath, "Contents", "MacOS", bundleName)
		}
		cmd = exec.Command("/bin/sh", "-c", `ELECTRON_ENABLE_LOGGING=1 exec "$0"`, innerExe)
	} else {
		cmd = exec.Command(exePath)
		cmd.Dir = path.Dir(exePath)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		a.Logger.Error("Failed to launch game", err)
		return fmt.Errorf("failed to launch game: %w", err)
	}

	a.gameMu.Lock()
	a.gameCmd = cmd
	a.gameMu.Unlock()

	wailsruntime.EventsEmit(a.ctx, "game:status", "running")

	// Stream stdout/stderr to frontend
	go a.streamGameOutput(stdout, "stdout")
	go a.streamGameOutput(stderr, "stderr")

	// Wait for process exit in background
	go func() {
		err := cmd.Wait()
		a.gameMu.Lock()
		a.gameCmd = nil
		a.gameMu.Unlock()

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
			a.Logger.Warn("Game exited with error", "error", err)
		} else {
			a.Logger.Info("Game exited normally")
		}
		wailsruntime.EventsEmit(a.ctx, "game:exit", exitCode)
		wailsruntime.EventsEmit(a.ctx, "game:status", "stopped")
	}()

	return nil
}

func (a *App) streamGameOutput(r io.Reader, stream string) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		wailsruntime.EventsEmit(a.ctx, "game:log", map[string]string{
			"stream": stream,
			"line":   line,
		})
	}
}

func (a *App) IsGameRunning() bool {
	a.gameMu.Lock()
	defer a.gameMu.Unlock()
	return a.gameCmd != nil && a.gameCmd.ProcessState == nil
}

func (a *App) StopGame() error {
	a.gameMu.Lock()
	cmd := a.gameCmd
	a.gameMu.Unlock()

	if cmd == nil || cmd.ProcessState != nil {
		return fmt.Errorf("game is not running")
	}

	if a.pmtilesServer != nil {
		a.pmtilesServer.Close()
	}

	return cmd.Process.Kill()
}

func (a *App) ManuallyCheckForUpdates() {
	a.Logger.Info("Manually checking for updates")
	updater.CheckForUpdates(a.ctx, a.Downloader.OnProgress, a.Logger)
}

func (a *App) GetCurrentVersion() string {
	return strings.ToValidUTF8(constants.RAILYARD_VERSION, "")
}

func (a *App) startPMTilesServer() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		a.Logger.Warn("Failed to start PMTiles server listener", "error", err)
		return -1, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	a.Logger.Info(fmt.Sprintf("Starting PMTiles server on port %d", port))

	channel := make(chan error, 1)

	go func(l *logger.AppLogger, port int, errorChan chan error) {
		pmtilesServer, err := pmtiles.NewServerWithBucket(pmtiles.NewFileBucket(path.Join(paths.AppDataRoot(), "tiles")), "", log.New(l.Writer, "pmtiles: ", log.Default().Flags()), 128, "")
		if err != nil {
			l.Error("Failed to create PMTiles server", err)
			errorChan <- err
			return
		}
		pmtilesServer.Start()

		thumbnailDir := a.Config.Cfg.MetroMakerDataPath
		if thumbnailDir != "" {
			thumbnailDir = path.Join(thumbnailDir, "public", "data", "city-maps")
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/thumbnails/", func(w http.ResponseWriter, r *http.Request) {
			filePath := path.Join(thumbnailDir, path.Base(r.URL.Path))
			w.Header().Set("Access-Control-Allow-Origin", "*")
			http.ServeFile(w, r, filePath)
		})
		mux.HandleFunc("/debug/thumbnails", func(w http.ResponseWriter, r *http.Request) {
			entries, err := os.ReadDir(thumbnailDir)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err != nil {
				fmt.Fprintf(w, "<html><body><h1>Error</h1><pre>%s</pre></body></html>", err.Error())
				return
			}
			fmt.Fprint(w, `<html><head><style>
				body { font-family: monospace; background: #1a1a2e; color: #e0e0e0; padding: 2rem; }
				h1 { color: #fff; }
				a { color: #7c9bff; display: block; margin: 0.5rem 0; }
				img { max-width: 200px; max-height: 200px; border: 1px solid #333; margin: 0.5rem; }
				.entry { display: inline-block; text-align: center; margin: 1rem; }
			</style></head><body>`)
			fmt.Fprintf(w, "<h1>Thumbnails (%d)</h1>", len(entries))
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				url := fmt.Sprintf("http://127.0.0.1:%d/thumbnails/%s", port, e.Name())
				fmt.Fprintf(w, `<div class="entry"><a href="%s"><img src="%s" /><br/>%s</a></div>`, url, url, e.Name())
			}
			fmt.Fprint(w, "</body></html>")
		})
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			statusCode := pmtilesServer.ServeHTTP(w, r)
			l.Info("Handled PMTiles request", "path", r.URL.Path, "status", statusCode)
		})
		errorChan <- nil
		a.pmtilesServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		}
		l.Error("PMTiles error: ", a.pmtilesServer.ListenAndServe())
	}(a.Logger, port, channel)
	return port, <-channel
}

func (a *App) generateMissingThumbnails(port int) {
	thumbnailDir := path.Join(a.Config.Cfg.MetroMakerDataPath, "public", "data", "city-maps")
	os.MkdirAll(thumbnailDir, os.ModePerm)

	for _, m := range a.Registry.GetInstalledMaps() {
		svgPath := path.Join(thumbnailDir, m.MapConfig.Code+".svg")
		if _, err := os.Stat(svgPath); err == nil {
			continue
		}
		if m.MapConfig.ThumbnailBbox == nil && m.MapConfig.Bbox == nil && m.MapConfig.InitialViewState.Latitude == 0 && m.MapConfig.InitialViewState.Longitude == 0 {
			continue
		}
		a.Logger.Info("Generating missing thumbnail", "map", m.MapConfig.Code)
		data, err := utils.GenerateThumbnail(m.MapConfig.Code, m.MapConfig, port)
		if err != nil {
			a.Logger.Warn("Failed to generate thumbnail", "map", m.MapConfig.Code, "error", err)
			continue
		}
		if err := os.WriteFile(svgPath, []byte(data), 0644); err != nil {
			a.Logger.Warn("Failed to save thumbnail", "map", m.MapConfig.Code, "error", err)
		}
	}
}

func (a *App) generateMod(port int) error {
	maps := a.Registry.GetInstalledMaps()
	a.Logger.Info("Generating mod with maps", "count", len(maps))
	places := make([]types.ConfigData, 0, len(maps))
	for _, m := range maps {
		places = append(places, m.MapConfig)
	}
	config := types.MetroMakerModConfig{
		Port:          port,
		TileZoomLevel: 15,
		Places:        places,
	}
	manifest := types.MetroMakerModManifest{
		Id:          "com.railyard.maploader",
		Name:        "Railyard Map Loader",
		Description: "Loads any custom maps installed by Railyard.",
		Version:     strings.Replace(constants.MOD_VERSION, "v", "", 1),
		Author: struct {
			Name string `json:"name"`
		}{
			Name: "Railyard",
		},
		Main: "index.js",
		Dependencies: map[string]string{
			constants.GameDependencyKey: ">=1.0.0",
		},
	}
	stringifiedConfig, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal mod config: %w", err)
	}
	modContent := constants.ModTemplateWithConfig(string(stringifiedConfig))
	manifestContent, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal mod manifest: %w", err)
	}
	modsFolder := path.Join(a.Config.Cfg.MetroMakerDataPath, "mods", "mapLoader")
	if err := os.MkdirAll(modsFolder, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create mod directory: %w", err)
	}

	if err := os.WriteFile(path.Join(modsFolder, "index.js"), []byte(modContent), 0644); err != nil {
		return fmt.Errorf("failed to write mod index.js: %w", err)
	}

	if err := os.WriteFile(path.Join(modsFolder, "manifest.json"), manifestContent, 0644); err != nil {
		return fmt.Errorf("failed to write mod manifest.json: %w", err)
	}
	return nil
}
