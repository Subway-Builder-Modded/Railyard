import { useEffect, useState } from 'react';
import { Route, Switch, useLocation } from 'wouter';

import { DownloadNotification } from '@/components/layout/DownloadNotification';
import { Layout } from '@/components/layout/Layout';
import { MultiStepLoader } from '@/components/layout/MultiStepLoader';
import { SetupScreen } from '@/components/setup/SetupScreen';
import { Toaster } from '@/components/ui/sonner';
import { TooltipProvider } from '@/components/ui/tooltip';
import { useTheme } from '@/hooks/use-theme';
import { BrowsePage } from '@/pages/BrowsePage';
import { HomePage } from '@/pages/HomePage';
import { LibraryPage } from '@/pages/LibraryPage';
import { LogsPage } from '@/pages/LogsPage';
import { ProjectPage } from '@/pages/ProjectPage';
import { SettingsPage } from '@/pages/SettingsPage';
import { useConfigStore } from '@/stores/config-store';
import { useGameStore } from '@/stores/game-store';
import { useInstalledStore } from '@/stores/installed-store';
import { useProfileStore } from '@/stores/profile-store';
import { useRegistryStore } from '@/stores/registry-store';

import { ConsumePendingDeepLink, IsStartupReady } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { ExtractNotification } from './components/layout/ExtractNotification';

interface DownloadCancelledEvent {
  itemId?: string;
}

interface DeepLinkEvent {
  type?: string;
  id?: string;
}

function App() {
  useTheme();
  const [, setLocation] = useLocation();
  const [startupReady, setStartupReady] = useState(false);
  const [pendingDeepLinkRoute, setPendingDeepLinkRoute] = useState<
    string | null
  >(null);
  const updateInstalledLists = useInstalledStore((s) => s.updateInstalledLists);
  const acknowledgeCancel = useInstalledStore(
    (s) => s.acknowledgeCancelledInstall,
  );

  const initConfig = useConfigStore((s) => s.initialize);
  const configInitialized = useConfigStore((s) => s.initialized);
  const isConfigured = useConfigStore(
    (s) => s.validation?.isConfigured ?? false,
  );
  const setupCompleted = useConfigStore(
    (s) => s.config?.setupCompleted ?? false,
  );

  const initProfile = useProfileStore((s) => s.initialize);

  const initRegistry = useRegistryStore((s) => s.initialize);
  const registryInitialized = useRegistryStore((s) => s.initialized);
  const initInstalled = useInstalledStore((s) => s.initialize);
  const installedInitialized = useInstalledStore((s) => s.initialized);
  const profileInitialized = useProfileStore((s) => s.initialized);
  const initGame = useGameStore((s) => s.initialize);

  useEffect(() => {
    const registryUpdate = EventsOn('registry:update', () => {
      updateInstalledLists();
    });
    const downloadCancelled = EventsOn(
      'download:cancelled',
      (payload: DownloadCancelledEvent) => {
        if (!payload?.itemId) {
          return;
        }
        acknowledgeCancel(payload.itemId);
      },
    );
    const deepLinkOpened = EventsOn(
      'deeplink:open',
      (payload: DeepLinkEvent) => {
        const routeType = payload?.type;
        const routeID = payload?.id;
        if (!routeType || !routeID) {
          return;
        }
        setPendingDeepLinkRoute(
          `/project/${routeType}/${encodeURIComponent(routeID)}`,
        );
      },
    );
    let cancelled = false;
    let timer: number | undefined;

    const pollStartupReady = async () => {
      try {
        const readyResponse = await IsStartupReady();
        if (cancelled) return;
        if (readyResponse.status === 'success' && readyResponse.ready) {
          setStartupReady(true);
          return;
        }
      } catch {
        // Keep polling while backend startup is still in progress.
      }

      if (!cancelled) {
        timer = window.setTimeout(pollStartupReady, 250);
      }
    };

    pollStartupReady();

    return () => {
      registryUpdate();
      downloadCancelled();
      deepLinkOpened();
      cancelled = true;
      if (timer !== undefined) {
        window.clearTimeout(timer);
      }
    };
  }, [updateInstalledLists, acknowledgeCancel]);

  // Phase 1: config + profile + game events
  useEffect(() => {
    if (!startupReady) return;
    initConfig();
    initProfile();
    initGame();
  }, [startupReady, initConfig, initProfile, initGame]);

  // Phase 2: registry + installed (only when configured)
  useEffect(() => {
    if (startupReady && configInitialized && isConfigured) {
      initRegistry();
      initInstalled();
    }
  }, [
    startupReady,
    configInitialized,
    isConfigured,
    initRegistry,
    initInstalled,
  ]);

  // Build loading states based on current initialization progress
  const showRegistrySteps = configInitialized && isConfigured && setupCompleted;
  const loadingStates = [
    { text: 'Starting backend services' },
    { text: 'Loading configuration' },
    { text: 'Applying theme preferences' },
    { text: 'Loading user profile' },
    ...(showRegistrySteps
      ? [
          { text: 'Connecting to registry' },
          { text: 'Loading installed content' },
        ]
      : []),
  ];

  let currentStep = 0;
  if (startupReady) currentStep = 1;
  if (startupReady && configInitialized) currentStep = 2;
  if (startupReady && configInitialized) currentStep = 3;
  if (startupReady && configInitialized && profileInitialized) {
    currentStep = 3;
    if (showRegistrySteps) {
      currentStep = 4;
      if (registryInitialized) currentStep = 5;
      if (registryInitialized && installedInitialized) currentStep = 6;
    }
  }

  const baseLoading =
    !startupReady || !configInitialized || !profileInitialized;
  const registryLoading =
    showRegistrySteps && (!registryInitialized || !installedInitialized);

  useEffect(() => {
    if (!startupReady) return;

    ConsumePendingDeepLink()
      .then((response) => {
        if (response.status !== 'success') {
          return;
        }
        const routeType = response.target?.type;
        const routeID = response.target?.id;
        if (!routeType || !routeID) {
          return;
        }
        setPendingDeepLinkRoute(
          `/project/${routeType}/${encodeURIComponent(routeID)}`,
        );
      })
      .catch(() => {});
  }, [startupReady]);

  useEffect(() => {
    if (baseLoading || registryLoading || !isConfigured || !setupCompleted) {
      return;
    }
    if (!pendingDeepLinkRoute) {
      return;
    }

    setLocation(pendingDeepLinkRoute);
    setPendingDeepLinkRoute(null);
  }, [
    baseLoading,
    registryLoading,
    isConfigured,
    pendingDeepLinkRoute,
    setLocation,
    setupCompleted,
  ]);

  if (baseLoading || registryLoading) {
    return (
      <div className="railyard-accent">
        <MultiStepLoader
          loadingStates={loadingStates}
          currentStep={currentStep}
        />
      </div>
    );
  }

  // Gate: show setup if not configured OR setup not completed
  if (!isConfigured || !setupCompleted) {
    return (
      <div className="railyard-accent">
        <SetupScreen />
        <Toaster />
      </div>
    );
  }

  return (
    <div className="railyard-accent">
      <TooltipProvider>
        <Layout>
          <Switch>
            <Route path="/" component={HomePage} />
            <Route path="/library" component={LibraryPage} />
            <Route path="/browse" component={BrowsePage} />
            <Route path="/search" component={BrowsePage} />
            <Route path="/project/:type/:id" component={ProjectPage} />
            <Route path="/logs" component={LogsPage} />
            <Route path="/settings" component={SettingsPage} />
          </Switch>
        </Layout>
        <DownloadNotification />
        <ExtractNotification />
        <Toaster />
      </TooltipProvider>
    </div>
  );
}

export default App;
