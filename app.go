package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"railyard/internal/config"
	"railyard/internal/constants"
	"railyard/internal/downloader"
	"railyard/internal/files"
	"railyard/internal/logger"
	"railyard/internal/paths"
	"railyard/internal/profiles"
	"railyard/internal/registry"
	"railyard/internal/types"
	"railyard/internal/updater"
	"railyard/internal/utils"

	"github.com/beescuit/asar"
	"github.com/protomaps/go-pmtiles/pmtiles"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func panik(message string) {
	panic(message)
}

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

	deepLinks deepLinkQueue

	cachedGameVersion types.GameVersionResponse
}

// NewApp creates a new App application struct
func NewApp() *App {
	l := logger.NewAppLogger()
	cfg := config.NewConfig(l)
	reg := registry.NewRegistry(l, cfg)
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
	a.Config.SetContext(ctx)
	a.Downloader.InstallDependency = func(itemId string, itemType types.AssetType, version types.Version) {
		result := a.Profiles.UpdateSubscriptions(types.UpdateSubscriptionsRequest{
			ProfileID:             a.Profiles.GetActiveProfile().Profile.ID,
			Action:                types.SubscriptionActionSubscribe,
			ApplyMode:             types.UpdateSubscriptionsPersistOnly,
			SkipDependencyInstall: true,
			Assets: map[string]types.SubscriptionUpdateItem{
				itemId: {
					Version: version,
					Type:    itemType,
					IsLocal: false,
				},
			},
		})
		if result.Status == types.ResponseError {
			a.Logger.MultipleError("Failed to persist dependency subscription", logger.AsErrors(result.Errors), "item_id", itemId, "item_type", itemType, "version", version)
		}
	}

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
	a.Downloader.OnCancelled = func(itemId string, assetType types.AssetType, phase string) {
		wailsruntime.EventsEmit(ctx, "download:cancelled", map[string]interface{}{
			"itemId":    itemId,
			"assetType": string(assetType),
			"phase":     phase,
		})
	}
	a.Downloader.GetGameVersion = func() types.GameVersionResponse {
		return a.GetGameVersion()
	}
	a.Downloader.OnRegistryUpdate = func() {
		wailsruntime.EventsEmit(ctx, "registry:update")
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
	a.configureTmpStagingRoots()
	a.cleanupTmpStaging("startup")

	activeProfile := resolveStartupProfile(a)
	a.Logger.Info("Active user profile loaded on startup", "profile_id", activeProfile.ID)

	if err := a.Registry.Initialize(); err != nil {
		a.Logger.Warn("Failed to ensure local registry repository", "error", err)
	} else {
		a.bootstrapInstalledState(activeProfile)
	}
	if err := a.addSaltsOnFirstRun(); err != nil {
		a.Logger.Warn("Failed to add salts to existing assets on first run", "error", err)
	}
	if a.Config.Cfg.CheckForUpdatesOnLaunch {
		updater.CheckForUpdates(a.ctx, a.Downloader.OnProgress, a.Logger, a.Config.GetGithubToken())
	}

	// Registry must be initialized + startup profile ready so that initial Frontend state is viable.
	a.setStartupReady(true)
	// Keep startup deep links queued until the frontend consumes them.
	// Emitting here can race listener registration during cold launches.
	go runNonBlockingStartupRoutines(a, activeProfile)
}

func (a *App) setStartupReady(ready bool) {
	a.startupMu.Lock()
	defer a.startupMu.Unlock()
	a.startupReady = ready
}

// IsStartupReady reports whether backend startup routines have completed.
func (a *App) IsStartupReady() types.StartupReadyResponse {
	a.startupMu.RLock()
	defer a.startupMu.RUnlock()
	return types.StartupReadyResponse{
		GenericResponse: types.SuccessResponse("Startup status resolved"),
		Ready:           a.startupReady,
	}
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

	if result := a.Config.SaveConfig(); result.Status == types.ResponseError {
		log.Printf("Warning: failed to save config on shutdown: %s", result.Message)
	}
	if err := a.Registry.WriteInstalledToDisk(); err != nil {
		log.Printf("Warning: failed to persist installed registry state on shutdown: %v", err)
	}

	res := a.StopGame()
	if res.Status == types.ResponseError {
		log.Printf("Warning: failed to stop game on shutdown: %s", res.Message)
	}

	a.cleanupTmpStaging("shutdown")
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
	wailsruntime.EventsOn(a.ctx, "deeplink:start-game", func(optionalData ...interface{}) {
		if a.gameCmd != nil && a.gameCmd.ProcessState == nil {
			return
		}
		a.LaunchGame()
	})
	wailsruntime.WindowMaximise(a.ctx)
	if activeProfile.SystemPreferences.RefreshRegistryOnStartup {
		if err := a.Registry.Refresh(); err != nil {
			a.Logger.Warn("Failed to refresh registry on startup", "error", err)
		}
	}

	// Sync subscriptions for active profile on startup
	// TODO: Make this configurable within the profile itself
	syncResult := a.Profiles.SyncSubscriptions(activeProfile.ID, false, false)
	switch syncResult.Status {
	case types.ResponseError:
		a.Logger.MultipleError("Failed to sync profile subscriptions on startup", logger.AsErrors(syncResult.Errors), "profile_id", activeProfile.ID)
	case types.ResponseWarn:
		a.Logger.Warn("Profile subscriptions synced with warnings on startup", "message", syncResult.Message, "profile_id", activeProfile.ID, "error_count", len(syncResult.Errors))
	}
}

func (a *App) bootstrapInstalledState(activeProfile types.UserProfile) {
	err := a.Registry.BootstrapInstalledStateFromProfile(activeProfile)
	if err != nil {
		// This should not be blocking as we are already in an error state
		a.Logger.Error("Failed to bootstrap installed asset state on startup", err, "profile_id", activeProfile.ID)
	}
}

func (a *App) configureTmpStagingRoots() {
	cfgResult := a.Config.GetConfig()
	// Set separate staging roots for app data and MetroMaker data to allow atomic operations (and avoid cross-drive issues) for both locations
	roots := []files.StagingRoot{
		{
			TargetRoot:  paths.AppDataRoot(),
			StagingRoot: paths.AppTmpStagingPath(),
		},
	}
	if cfgResult.Config.MetroMakerDataPath != "" {
		roots = append(roots, files.StagingRoot{
			TargetRoot:  cfgResult.Config.MetroMakerDataPath,
			StagingRoot: paths.MetroMakerTmpStagingPath(cfgResult.Config.MetroMakerDataPath),
		})
	}
	files.ConfigureTmpStagingRoots(roots)
}

func (a *App) cleanupTmpStaging(stage string) {
	if err := files.CleanupTmpStagingRoots(); err != nil {
		a.Logger.Warn("Failed to cleanup managed atomic staging directories", "stage", stage, "error", err)
	}
}

// GetGameVersion attempts to detect the installed Subway Builder version.
// Returns an empty version with a warning status if detection fails.
func (a *App) GetGameVersion() types.GameVersionResponse {
	a.Logger.Info("Attempting to resolve game version")
	if a.cachedGameVersion != (types.GameVersionResponse{}) {
		a.Logger.Info("Returning cached game version", "version", a.cachedGameVersion.Version)
		return a.cachedGameVersion
	}
	cfg := a.Config.GetConfig()
	if !cfg.Validation.ExecutablePathValid {
		return types.GameVersionResponse{
			GenericResponse: types.WarnResponse("Game version not detected"),
			Version:         "",
		}
	}
	exePath := cfg.Config.ExecutablePath

	var asarPath string
	if runtime.GOOS == "darwin" {
		asarPath = filepath.Join(exePath, "Contents", "Resources", "app.asar")
	} else {
		asarPath = filepath.Join(filepath.Dir(exePath), "resources", "app.asar")
	}

	archiveFile, err := os.Open(asarPath)
	if err != nil {
		a.Logger.Warn("Failed to open app.asar for game version detection", "error", err, "asarPath", asarPath)
		a.cachedGameVersion = types.GameVersionResponse{
			GenericResponse: types.WarnResponse("Game version not detected"),
			Version:         "",
		}
		return types.GameVersionResponse{
			GenericResponse: types.WarnResponse("Game version not detected"),
			Version:         "",
		}
	}

	archive, err := asar.Decode(archiveFile)
	if err != nil {
		a.Logger.Warn("Failed to decode app.asar for game version detection", "error", err, "asarPath", asarPath)
		a.cachedGameVersion = types.GameVersionResponse{
			GenericResponse: types.WarnResponse("Game version not detected"),
			Version:         "",
		}
		return types.GameVersionResponse{
			GenericResponse: types.WarnResponse("Game version not detected"),
			Version:         "",
		}
	}

	packageFile := archive.Find("package.json")
	if packageFile == nil {
		a.Logger.Warn("Failed to find package.json in app.asar", "asarPath", asarPath)
		a.cachedGameVersion = types.GameVersionResponse{
			GenericResponse: types.WarnResponse("Game version not detected"),
			Version:         "",
		}
		return types.GameVersionResponse{
			GenericResponse: types.WarnResponse("Game version not detected"),
			Version:         "",
		}
	}

	packageReader := packageFile.Open()
	var pkg struct {
		Version string `json:"version"`
	}

	if err := json.NewDecoder(packageReader).Decode(&pkg); err != nil {
		a.Logger.Warn("Failed to decode package.json", "asarPath", asarPath, "error", err)
		a.cachedGameVersion = types.GameVersionResponse{
			GenericResponse: types.WarnResponse("Game version not detected"),
			Version:         "",
		}
		return types.GameVersionResponse{
			GenericResponse: types.WarnResponse("Game version not detected"),
			Version:         "",
		}
	}

	a.Logger.Info("Game version detected", "version", pkg.Version)
	resp := types.GameVersionResponse{
		GenericResponse: types.SuccessResponse("Game version detected"),
		Version:         pkg.Version,
	}
	a.cachedGameVersion = resp
	return resp
}

func (a *App) LaunchGame() types.GenericResponse {
	a.gameMu.Lock()
	if a.gameCmd != nil && a.gameCmd.ProcessState == nil {
		a.gameMu.Unlock()
		return types.ErrorResponse("game is already running")
	}
	a.gameMu.Unlock()

	cfg := a.Config.GetConfig()
	if !cfg.Validation.ExecutablePathValid {
		return types.ErrorResponse("game executable path is not configured or invalid")
	}

	extraSplitArgs := []string{}

	profile := a.Profiles.GetActiveProfile()
	if profile.Status != types.ResponseSuccess {
		a.Logger.Warn("Failed to get active profile for command line args on game launch", "status", profile.Status, "message", profile.Message)
	} else {
		if profile.Profile.SystemPreferences.ExtraHeapSize > 0 {
			extraSplitArgs = append(extraSplitArgs, fmt.Sprintf(`--js-flags="--max-old-space-size=%d"`, profile.Profile.SystemPreferences.ExtraHeapSize))
		}
	}

	port, err := a.startPMTilesServer()
	if err != nil {
		a.Logger.Warn("Failed to start PMTiles server", "error", err)
		return types.ErrorResponse(err.Error())
	}

	wailsruntime.EventsEmit(a.ctx, "server:port", port)
	a.Logger.Info(fmt.Sprintf("Debug thumbnails: http://127.0.0.1:%d/debug/thumbnails", port))

	a.generateMissingThumbnails(port)

	if err := a.generateMod(port); err != nil {
		a.Logger.Warn("Failed to generate mod", "error", err)
		return types.ErrorResponse(err.Error())
	}

	exePath := strings.TrimPrefix(cfg.Config.ExecutablePath, "/run/host") // Fix the paths when calling outside of sandbox
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
		args := []string{"-c", `ELECTRON_ENABLE_LOGGING=1 exec "$0" "$@"`, innerExe}
		args = append(args, extraSplitArgs...)
		cmd = exec.Command("/bin/sh", args...)
		if profile.Status == types.ResponseSuccess {
			if profile.Profile.SystemPreferences.UseDevTools {
				cmd.Env = append(os.Environ(), "DEBUG_PROD=TRUE")
			}
		}
	} else if runtime.GOOS == "linux" {
		// Prefer host launch via Flatpak
		if _, lookPathErr := exec.LookPath("flatpak-spawn"); lookPathErr == nil {
			if a.Config.Cfg.ChromeSandboxPath != "" {
				// Ensure sandbox is used if available to avoid permission issues in Flatpak environments
				args := []string{"--env=CHROME_DEVEL_SANDBOX=" + a.Config.Cfg.ChromeSandboxPath, "--host", exePath}
				args = append(args, extraSplitArgs...)
				cmd = exec.Command("flatpak-spawn", args...)
			} else {
				args := []string{"--host", exePath, "--no-sandbox"}
				args = append(args, extraSplitArgs...)
				cmd = exec.Command("flatpak-spawn", args...)
			}
		} else {
			// Fall back to direct launch if flatpak-spawn is not available
			a.Logger.Warn("flatpak-spawn not available; falling back to direct executable launch", "error", lookPathErr)
			cmd = exec.Command(exePath, extraSplitArgs...)
			cmd.Dir = filepath.Dir(exePath)
		}
		if profile.Status == types.ResponseSuccess {
			if profile.Profile.SystemPreferences.UseDevTools {
				cmd.Env = append(os.Environ(), "DEBUG_PROD=TRUE")
			}
		}
	} else {
		cmd = exec.Command(exePath, extraSplitArgs...)
		cmd.Dir = filepath.Dir(exePath)

		if profile.Status == types.ResponseSuccess {
			if profile.Profile.SystemPreferences.UseDevTools {
				cmd.Env = append(os.Environ(), "DEBUG_PROD=TRUE")
			}
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return types.ErrorResponse(fmt.Sprintf("failed to create stdout pipe: %v", err))
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return types.ErrorResponse(fmt.Sprintf("failed to create stderr pipe: %v", err))
	}

	if err := cmd.Start(); err != nil {
		a.Logger.Error("Failed to launch game", err)
		return types.ErrorResponse(fmt.Sprintf("failed to launch game: %v", err))
	}

	a.gameMu.Lock()
	a.gameCmd = cmd
	a.gameMu.Unlock()

	wailsruntime.EventsEmit(a.ctx, "game:status", "running")

	wailsruntime.EventsEmit(a.ctx, "game:log", map[string]string{
		"stream": "stdout",
		"line":   fmt.Sprintf("> %s %s", strings.Split(a.gameCmd.Path, string(os.PathSeparator))[len(strings.Split(a.gameCmd.Path, string(os.PathSeparator)))-1], strings.Join(a.gameCmd.Args[1:], " ")),
	})

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

	return types.SuccessResponse("Game launched")
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

func (a *App) IsGameRunning() types.GameRunningResponse {
	a.gameMu.Lock()
	defer a.gameMu.Unlock()
	return types.GameRunningResponse{
		GenericResponse: types.SuccessResponse("Game running status resolved"),
		Running:         a.gameCmd != nil && a.gameCmd.ProcessState == nil,
	}
}

func (a *App) StopGame() types.GenericResponse {
	a.Logger.Info("Killing game process")
	a.gameMu.Lock()
	cmd := a.gameCmd
	a.gameMu.Unlock()

	if cmd == nil || cmd.ProcessState != nil {
		a.Logger.Warn("No game process to kill")
		return types.ErrorResponse("game is not running")
	}

	if a.pmtilesServer != nil {
		a.Logger.Info("Shutting down PMTiles server")
		a.pmtilesServer.Close()
	}

	if err := cmd.Process.Kill(); err != nil {
		a.Logger.Warn("Failed to kill game process", "error", err)
		return types.ErrorResponse(fmt.Sprintf("failed to stop game: %v", err))
	}

	a.Logger.Info("Game process killed successfully")
	a.gameCmd = nil
	return types.SuccessResponse("Game stopped")
}

func (a *App) ManuallyCheckForUpdates() types.GenericResponse {
	a.Logger.Info("Manually checking for updates")
	updater.CheckForUpdates(a.ctx, a.Downloader.OnProgress, a.Logger, a.Config.GetGithubToken())
	return types.SuccessResponse("Update check started")
}

func (a *App) GetCurrentVersion() types.AppVersionResponse {
	version := strings.ToValidUTF8(constants.RAILYARD_VERSION, "")
	return types.AppVersionResponse{
		GenericResponse: types.SuccessResponse("App version resolved"),
		Version:         version,
	}
}

func (a *App) InstallLinuxSandbox() types.GenericResponse {
	a.Logger.Info("Installing Linux sandbox")
	if runtime.GOOS != "linux" {
		panic("InstalLinuxSandbox shouldn't be possible to call on a non-linux platform")
	}

	if a.Config.Cfg.ExecutablePath == "" {
		a.Logger.Warn("Game executable path is not configured, stopping.")
		return types.ErrorResponse("game executable path is not configured")
	}

	cmd := exec.Command(a.Config.Cfg.ExecutablePath, "--appimage-extract", "chrome-sandbox")
	cmd.Dir = "/tmp"
	err := cmd.Run()
	if err != nil {
		a.Logger.Error("Failed to extract chrome-sandbox using flatpak-spawn", err)
		return types.ErrorResponse(fmt.Sprintf("failed to extract chrome-sandbox: %v", err))
	}

	sandboxPath := path.Join("/tmp", "squashfs-root", "chrome-sandbox")
	if _, err := os.Stat(sandboxPath); errors.Is(err, fs.ErrNotExist) {
		a.Logger.Error("Extracted chrome-sandbox not found at expected path", err)
		return types.ErrorResponse(fmt.Sprintf("extracted chrome-sandbox not found at expected path: %s", sandboxPath))
	}

	destPath := path.Join("/usr", "local", "bin", "chrome-sb-sandbox")
	if _, lookPathErr := exec.LookPath("flatpak-spawn"); lookPathErr == nil {
		cmd = exec.Command("flatpak-spawn", "--host", "pkexec", "install", "-o", "root", "-g", "root", "-m", "4755", sandboxPath, destPath)
	} else {
		cmd = exec.Command("pkexec", "install", "-o", "root", "-g", "root", "-m", "4755", sandboxPath, destPath)
	}
	if err := cmd.Run(); err != nil {
		a.Logger.Error("Failed to install chrome-sandbox with pkexec", err)
		return types.ErrorResponse(fmt.Sprintf("failed to install chrome-sandbox with pkexec: %v", err))
	}
	a.Config.Cfg.ChromeSandboxPath = destPath
	return types.SuccessResponse("Linux sandbox installed")
}

func (a *App) SandboxIsInstalled() types.SandboxStatusResponse {
	installed := false
	if runtime.GOOS == "linux" && a.Config.Cfg.ChromeSandboxPath != "" {
		if _, err := os.Stat(a.Config.Cfg.ChromeSandboxPath); !errors.Is(err, fs.ErrNotExist) {
			installed = true
		}
	}

	return types.SandboxStatusResponse{
		GenericResponse: types.SuccessResponse("Sandbox status resolved"),
		Installed:       installed,
	}
}

func (a *App) GetPlatform() types.PlatformResponse {
	return types.PlatformResponse{
		GenericResponse: types.SuccessResponse("Platform resolved"),
		Platform:        runtime.GOOS,
	}
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

func (a *App) addSaltsOnFirstRun() error {
	if _, err := os.Stat(paths.JoinLocalPath(paths.AppDataRoot(), constants.RailyardAssetsSaltedMarker)); errors.Is(err, fs.ErrNotExist) {
		a.Logger.Info("Adding salts to existing assets on first run")
		for _, mod := range a.Registry.GetInstalledMods() {
			id := mod.ID

			if _, err := os.Create(paths.JoinLocalPath(a.Config.Cfg.GetModsFolderPath(), id, constants.RailyardAssetMarker)); err != nil {
				a.Logger.Warn("Failed to add salt file for mod", "mod_id", id, "error", err)
				return err
			}
		}

		for _, m := range a.Registry.GetInstalledMaps() {
			code := m.MapConfig.Code
			if _, err := os.Create(paths.JoinLocalPath(a.Config.Cfg.GetMapsFolderPath(), code, constants.RailyardAssetMarker)); err != nil {
				a.Logger.Warn("Failed to add salt file for map", "map_code", code, "error", err)
				return err
			}
		}

		// Create a marker file to indicate that salts have been added, so we don't repeat this process on subsequent runs
		if _, err := os.Create(paths.JoinLocalPath(paths.AppDataRoot(), constants.RailyardAssetsSaltedMarker)); err != nil {
			a.Logger.Warn("Failed to create asset salted marker file", "error", err)
			return err
		}
	}
	return nil
}

func (a *App) GetTotalMemory() (uint64, error) {
	return utils.GetTotalSystemMemoryMB()
}
