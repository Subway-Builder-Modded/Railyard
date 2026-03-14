package downloader

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"railyard/internal/config"
	"railyard/internal/logger"
	"railyard/internal/registry"
	"railyard/internal/testutil/registrytest"
	"railyard/internal/types"

	"github.com/stretchr/testify/require"
)

type testRegistryLogSink struct{}

func (testRegistryLogSink) Info(string, ...any)         {}
func (testRegistryLogSink) Warn(string, ...any)         {}
func (testRegistryLogSink) Error(string, error, ...any) {}

func newTestDownloader() *Downloader {
	return &Downloader{}
}

func configureDownloaderConfig(t *testing.T, cfg *config.Config) {
	t.Helper()
	cfg.Cfg.MetroMakerDataPath = t.TempDir()
	exePath := filepath.Join(t.TempDir(), "subway-builder.exe")
	require.NoError(t, os.WriteFile(exePath, []byte("exe"), 0o644))
	cfg.Cfg.ExecutablePath = exePath
}

func runInParallel(total int, fn func(index int)) {
	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			fn(index)
		}(i)
	}
	wg.Wait()
}

func operationSuccess(message string, delay time.Duration, onRun func()) func() operationResult {
	return func() operationResult {
		if onRun != nil {
			onRun()
		}
		if delay > 0 {
			time.Sleep(delay)
		}
		return operationResult{
			genericResponse: types.GenericResponse{
				Status:  types.ResponseSuccess,
				Message: message,
			},
		}
	}
}

func supersededSuccess(message string) operationResult {
	return operationResult{
		genericResponse: types.GenericResponse{
			Status:  types.ResponseSuccess,
			Message: message,
		},
	}
}

func enqueueOperation(d *Downloader, action operationAction, assetType types.AssetType, assetID string, version string, run func() operationResult) operationResult {
	return d.enqueueOperation(
		downloadQueueKey{assetType: assetType, assetID: assetID},
		d.operationKey(action, assetType, assetID, version),
		run,
		supersededSuccess("Operation superseded by newer queued request. No action taken."),
	)
}

// waitForPendingOperation is a helper function that waits until the specified asset key is present in the downloader's pending map, or fails the test if it times out.
func waitForPendingOperation(t *testing.T, d *Downloader, assetKey downloadQueueKey) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		d.downloadMu.Lock()
		_, ok := d.pending[assetKey]
		d.downloadMu.Unlock()
		if ok {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for pending operation: %s", assetKey)
}

func updateAtomicMax(max *int32, current int32) {
	for {
		existing := atomic.LoadInt32(max)
		if current <= existing {
			return
		}
		if atomic.CompareAndSwapInt32(max, existing, current) {
			return
		}
	}
}

func TestEnqueueOperationSupersedesForSameAsset(t *testing.T) {
	d := newTestDownloader()

	blockerStarted := make(chan struct{})
	releaseBlocker := make(chan struct{})
	go func() {
		_ = enqueueOperation(d, operationActionInstall, types.AssetTypeMod, "blocker-mod", "1.0.0", func() operationResult {
			close(blockerStarted)
			<-releaseBlocker
			return operationResult{
				genericResponse: types.GenericResponse{
					Status:  types.ResponseSuccess,
					Message: "blocker complete",
				},
			}
		})
	}()
	<-blockerStarted

	var firstRunCount int32
	firstResultCh := make(chan operationResult, 1)
	go func() {
		firstResultCh <- enqueueOperation(d, operationActionInstall, types.AssetTypeMap, "map-a", "1.0.0", operationSuccess("first install ran", 0, func() {
			atomic.AddInt32(&firstRunCount, 1)
		}))
	}()
	waitForPendingOperation(t, d, downloadQueueKey{assetType: types.AssetTypeMap, assetID: "map-a"})

	var secondRunCount int32
	secondResultCh := make(chan operationResult, 1)
	go func() {
		secondResultCh <- enqueueOperation(d, operationActionInstall, types.AssetTypeMap, "map-a", "2.0.0", operationSuccess("second install ran", 0, func() {
			atomic.AddInt32(&secondRunCount, 1)
		}))
	}()

	firstResult := <-firstResultCh
	require.Equal(t, types.ResponseSuccess, firstResult.genericResponse.Status)
	require.True(t, strings.Contains(firstResult.genericResponse.Message, "superseded"))
	require.Equal(t, int32(0), atomic.LoadInt32(&firstRunCount))

	waitForPendingOperation(t, d, downloadQueueKey{assetType: types.AssetTypeMap, assetID: "map-a"})
	close(releaseBlocker)

	secondResult := <-secondResultCh
	require.Equal(t, types.ResponseSuccess, secondResult.genericResponse.Status)
	require.Equal(t, "second install ran", secondResult.genericResponse.Message)
	require.Equal(t, int32(1), atomic.LoadInt32(&secondRunCount))
}

func TestEnqueueOperationPreservesPendingQueuePosition(t *testing.T) {
	d := newTestDownloader()

	blockerStarted := make(chan struct{})
	releaseBlocker := make(chan struct{})
	go func() {
		_ = enqueueOperation(d, operationActionInstall, types.AssetTypeMod, "blocker-mod", "1.0.0", func() operationResult {
			close(blockerStarted)
			<-releaseBlocker
			return operationResult{
				genericResponse: types.GenericResponse{
					Status:  types.ResponseSuccess,
					Message: "blocker complete",
				},
			}
		})
	}()
	<-blockerStarted

	// Firsat install for map-a that will be superseded
	firstResultCh := make(chan operationResult, 1)
	go func() {
		firstResultCh <- enqueueOperation(d, operationActionInstall, types.AssetTypeMap, "map-a", "1.0.0", operationSuccess("first install", 0, nil))
	}()
	waitForPendingOperation(t, d, downloadQueueKey{assetType: types.AssetTypeMap, assetID: "map-a"})

	runOrder := make(chan string, 2)
	// Install for unrelated mod-b
	otherResultCh := make(chan operationResult, 1)
	go func() {
		otherResultCh <- enqueueOperation(d, operationActionInstall, types.AssetTypeMod, "mod-b", "1.0.0", operationSuccess("other install", 0, func() {
			runOrder <- "mod-b"
		}))
	}()
	waitForPendingOperation(t, d, downloadQueueKey{assetType: types.AssetTypeMod, assetID: "mod-b"})

	// Second install for map-a that should superseded the first install
	supersedingResultCh := make(chan operationResult, 1)
	go func() {
		supersedingResultCh <- enqueueOperation(d, operationActionInstall, types.AssetTypeMap, "map-a", "2.0.0", operationSuccess("second install", 0, func() {
			runOrder <- "map-a-2.0.0"
		}))
	}()

	firstResult := <-firstResultCh
	require.Equal(t, types.ResponseSuccess, firstResult.genericResponse.Status)
	require.True(t, strings.Contains(firstResult.genericResponse.Message, "superseded"))

	close(releaseBlocker)

	// Validate that the second install for map-a runs before the install for mod-b, as the initial install for map-a should have its position preserved during superseding
	firstRun := <-runOrder
	secondRun := <-runOrder
	require.Equal(t, "map-a-2.0.0", firstRun)
	require.Equal(t, "mod-b", secondRun)

	// Validate that the superseding result and other asset result are successful
	supersedingResult := <-supersedingResultCh
	require.Equal(t, types.ResponseSuccess, supersedingResult.genericResponse.Status)
	require.Equal(t, "second install", supersedingResult.genericResponse.Message)
	otherResult := <-otherResultCh
	require.Equal(t, types.ResponseSuccess, otherResult.genericResponse.Status)
	require.Equal(t, "other install", otherResult.genericResponse.Message)
}

func TestEnqueueOperationUsesLatestRequestForPendingAsset(t *testing.T) {
	d := newTestDownloader()

	blockerStarted := make(chan struct{})
	releaseBlocker := make(chan struct{})

	go func() {
		_ = enqueueOperation(d, operationActionInstall, types.AssetTypeMod, "blocker-mod", "1.0.0", func() operationResult {
			close(blockerStarted)
			<-releaseBlocker
			return operationResult{genericResponse: types.GenericResponse{Status: types.ResponseSuccess, Message: "blocker complete"}}
		})
	}()
	<-blockerStarted

	var installRunCount int32
	installResultCh := make(chan operationResult, 1)
	go func() {
		installResultCh <- enqueueOperation(d, operationActionInstall, types.AssetTypeMap, "map-a", "1.0.0", operationSuccess("install ran", 0, func() {
			atomic.AddInt32(&installRunCount, 1)
		}))
	}()
	waitForPendingOperation(t, d, downloadQueueKey{assetType: types.AssetTypeMap, assetID: "map-a"})

	var uninstallRunCount int32
	uninstallResultCh := make(chan operationResult, 1)
	go func() {
		uninstallResultCh <- enqueueOperation(d, operationActionUninstall, types.AssetTypeMap, "map-a", "", operationSuccess("uninstall ran", 0, func() {
			atomic.AddInt32(&uninstallRunCount, 1)
		}))
	}()

	installResult := <-installResultCh
	require.Equal(t, types.ResponseSuccess, installResult.genericResponse.Status)
	require.True(t, strings.Contains(installResult.genericResponse.Message, "superseded"))
	require.Equal(t, int32(0), atomic.LoadInt32(&installRunCount))

	waitForPendingOperation(t, d, downloadQueueKey{assetType: types.AssetTypeMap, assetID: "map-a"})
	close(releaseBlocker)

	uninstallResult := <-uninstallResultCh
	require.Equal(t, types.ResponseSuccess, uninstallResult.genericResponse.Status)
	require.Equal(t, "uninstall ran", uninstallResult.genericResponse.Message)
	require.Equal(t, int32(1), atomic.LoadInt32(&uninstallRunCount))
}

func TestEnqueueOperationRunsSequentially(t *testing.T) {
	d := newTestDownloader()
	const jobs = 5

	var runCount int32
	var activeCount int32
	var maxConcurrent int32

	runInParallel(jobs, func(i int) {
		result := enqueueOperation(
			d,
			operationActionInstall,
			types.AssetTypeMod,
			fmt.Sprintf("mod-%d", i),
			"1.0.0",
			operationSuccess("done", 20*time.Millisecond, func() {
				atomic.AddInt32(&runCount, 1)
				current := atomic.AddInt32(&activeCount, 1)
				updateAtomicMax(&maxConcurrent, current)
				defer atomic.AddInt32(&activeCount, -1)
			}),
		)
		require.Equal(t, types.ResponseSuccess, result.genericResponse.Status)
	})

	require.Equal(t, int32(jobs), atomic.LoadInt32(&runCount))
	require.Equal(t, int32(1), atomic.LoadInt32(&maxConcurrent))
}

func TestInstallMapForExistingIsNoOp(t *testing.T) {
	cfg := config.NewConfig()
	reg := registry.NewRegistry(testRegistryLogSink{}, cfg)
	expectedConfig := types.ConfigData{
		Code:        "ABC",
		Name:        "Map A",
		Description: "desc",
		Creator:     "tester",
		Version:     "1.0.0",
	}
	reg.AddInstalledMap("map-a", "1.0.0", expectedConfig)

	d := &Downloader{
		Registry: reg,
		Config:   cfg,
		Logger:   logger.LoggerAtPath(""),
	}

	// Validate that no-op response is returned at invocation time
	response := d.InstallAsset(types.AssetTypeMap, "map-a", "1.0.0")
	require.Equal(t, types.ResponseWarn, response.Status)
	require.Contains(t, response.Message, "already installed at requested version")
	require.Equal(t, expectedConfig, response.Config)
}

func TestInstallModPreservesNoOpThroughStateMutation(t *testing.T) {
	cfg := config.NewConfig()
	reg := registry.NewRegistry(testRegistryLogSink{}, cfg)
	d := &Downloader{
		Registry: reg,
		Config:   cfg,
		Logger:   logger.LoggerAtPath(""),
	}

	blockerStarted := make(chan struct{})
	releaseBlocker := make(chan struct{})

	// Enqueue an install operation for map-a that will block until released
	go func() {
		_ = enqueueOperation(d, operationActionInstall, types.AssetTypeMap, "blocker-map", "1.0.0", func() operationResult {
			close(blockerStarted)
			<-releaseBlocker
			return operationResult{genericResponse: types.GenericResponse{Status: types.ResponseSuccess, Message: "blocker complete"}}
		})
	}()
	<-blockerStarted

	responseCh := make(chan types.AssetInstallResponse, 1)
	// Enqueue an install operation for mod-a
	go func() {
		responseCh <- d.InstallAsset(types.AssetTypeMod, "mod-a", "1.0.0")
	}()

	waitForPendingOperation(t, d, downloadQueueKey{assetType: types.AssetTypeMod, assetID: "mod-a"})
	// Mutate registry state to make it appear as though mod-a is already installed while the install operation is still pending
	reg.AddInstalledMod("mod-a", "1.0.0")
	close(releaseBlocker)

	// Validate that no-op response is returned at execution time
	response := <-responseCh
	require.Equal(t, types.ResponseWarn, response.Status)
	require.Contains(t, response.Message, "already installed at requested version")
}

func TestInstallAssetError(t *testing.T) {
	testCases := []struct {
		name              string
		assetType         types.AssetType
		assetID           string
		version           string
		setup             func(t *testing.T, d *Downloader, reg *registry.Registry) func()
		expectedStatus    types.Status
		expectedErrorCode types.DownloaderErrorType
	}{
		{
			name:              "Invalid asset type",
			assetType:         types.AssetType("bad"),
			assetID:           "asset-a",
			version:           "1.0.0",
			expectedStatus:    types.ResponseError,
			expectedErrorCode: types.InstallErrorInvalidAssetType,
		},
		{
			name:              "Invalid config",
			assetType:         types.AssetTypeMod,
			assetID:           "mod-a",
			version:           "1.0.0",
			expectedStatus:    types.ResponseError,
			expectedErrorCode: types.InstallErrorInvalidConfig, // No config initialized
		},
		{
			name:      "Registry lookup failed",
			assetType: types.AssetTypeMod,
			assetID:   "missing-mod",
			version:   "1.0.0",
			setup: func(t *testing.T, d *Downloader, _ *registry.Registry) func() {
				t.Helper()
				configureDownloaderConfig(t, d.Config)
				return nil
			},
			expectedStatus:    types.ResponseError,
			expectedErrorCode: types.InstallErrorRegistryLookup,
		},
		{
			name:      "Version lookup failed",
			assetType: types.AssetTypeMap,
			assetID:   "map-a",
			version:   "1.0.0",
			setup: func(t *testing.T, d *Downloader, reg *registry.Registry) func() {
				t.Helper()
				configureDownloaderConfig(t, d.Config)
				return registrytest.MockRegistryServer(t, reg, []registrytest.UpdateFixture{
					{AssetID: "map-a", AssetType: types.AssetTypeMap, FailVersions: true, MapCode: "AAA"},
				})
			},
			expectedStatus:    types.ResponseError,
			expectedErrorCode: types.InstallErrorVersionLookup,
		},
		{
			name:      "Version not found",
			assetType: types.AssetTypeMod,
			assetID:   "mod-a",
			version:   "2.0.0",
			setup: func(t *testing.T, d *Downloader, reg *registry.Registry) func() {
				t.Helper()
				configureDownloaderConfig(t, d.Config)
				return registrytest.MockRegistryServer(t, reg, []registrytest.UpdateFixture{
					{AssetID: "mod-a", AssetType: types.AssetTypeMod, Versions: []string{"1.0.0"}},
				})
			},
			expectedStatus:    types.ResponseError,
			expectedErrorCode: types.InstallErrorVersionNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.NewConfig()
			reg := registry.NewRegistry(testRegistryLogSink{}, cfg)
			d := &Downloader{
				Registry: reg,
				Config:   cfg,
				Logger:   logger.LoggerAtPath(""),
			}
			d.tempPath = t.TempDir()
			d.mapTilePath = t.TempDir()

			var cleanup func()
			if tc.setup != nil {
				cleanup = tc.setup(t, d, reg)
			}
			if cleanup != nil {
				defer cleanup()
			}

			response := d.InstallAsset(tc.assetType, tc.assetID, tc.version)
			require.Equal(t, tc.expectedStatus, response.Status)
			require.Equal(t, tc.expectedErrorCode, response.ErrorType)
		})
	}
}

func TestInstallAssetSuccess(t *testing.T) {
	testCases := []struct {
		name          string
		assetType     types.AssetType
		assetID       string
		version       string
		fixtures      []registrytest.UpdateFixture
		expectedCode  string
		expectMapConf bool
	}{
		{
			name:      "Map install success",
			assetType: types.AssetTypeMap,
			assetID:   "map-a",
			version:   "1.0.0",
			fixtures: []registrytest.UpdateFixture{
				{AssetID: "map-a", AssetType: types.AssetTypeMap, Versions: []string{"1.0.0"}, MapCode: "AAA"},
			},
			expectedCode:  "AAA",
			expectMapConf: true,
		},
		{
			name:      "Mod install success",
			assetType: types.AssetTypeMod,
			assetID:   "mod-a",
			version:   "1.0.0",
			fixtures: []registrytest.UpdateFixture{
				{AssetID: "mod-a", AssetType: types.AssetTypeMod, Versions: []string{"1.0.0"}},
			},
			expectedCode:  "",    // No cityCode for mods
			expectMapConf: false, // No config for mod
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.NewConfig()
			reg := registry.NewRegistry(testRegistryLogSink{}, cfg)
			configureDownloaderConfig(t, cfg)
			d := &Downloader{
				Registry: reg,
				Config:   cfg,
				Logger:   logger.LoggerAtPath(""),
			}
			d.tempPath = t.TempDir()
			d.mapTilePath = t.TempDir()

			cleanup := registrytest.MockRegistryServer(t, reg, tc.fixtures)
			defer cleanup()

			response := d.InstallAsset(tc.assetType, tc.assetID, tc.version)
			require.Equal(t, types.ResponseSuccess, response.Status, response.Message)
			require.Equal(t, types.DownloaderErrorType(""), response.ErrorType)
			if tc.expectMapConf {
				require.Equal(t, tc.expectedCode, response.Config.Code)
			} else {
				require.Equal(t, types.ConfigData{}, response.Config)
			}
		})
	}
}

func TestIsValidOperationAction(t *testing.T) {
	require.True(t, isValidOperationAction(operationActionInstall))
	require.True(t, isValidOperationAction(operationActionUninstall))
	require.False(t, isValidOperationAction(operationAction("invalid")))
}

func TestOperationKeyPanicsOnInvalidAction(t *testing.T) {
	d := &Downloader{}
	require.Panics(t, func() {
		_ = d.operationKey(operationAction("invalid"), types.AssetTypeMap, "map-a", "1.0.0")
	})
}

func TestDownloadTempZipGithubAuthFallback(t *testing.T) {
	originalHostCheck := isGitHubDownloadHost
	isGitHubDownloadHost = func(string) bool { return true }
	defer func() { isGitHubDownloadHost = originalHostCheck }()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			// On first request, return a 403 on the Github token based request
			require.Equal(t, "Bearer ghp_test_token", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusForbidden)
			return
		}
		// Validate that an unauthenticated request is made on retry
		require.Empty(t, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("zip-content"))
	}))
	defer server.Close()

	cfg := config.NewConfig()
	cfg.Cfg.GithubToken = "ghp_test_token"
	d := &Downloader{
		Config:   cfg,
		Logger:   logger.LoggerAtPath(""),
		tempPath: t.TempDir(),
	}

	resp := d.downloadTempZip(server.URL+"/asset.zip", "asset-a")
	require.Equal(t, types.ResponseSuccess, resp.Status)
	require.NotEmpty(t, resp.Path)
	require.Equal(t, 2, requestCount)
}
