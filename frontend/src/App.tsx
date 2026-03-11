import { useEffect, useState } from "react";
import { Route, Switch } from "wouter";
import { TooltipProvider } from "@/components/ui/tooltip";
import { Toaster } from "@/components/ui/sonner";
import { Layout } from "@/components/layout/Layout";
import { SetupScreen } from "@/components/setup/SetupScreen";
import { MultiStepLoader } from "@/components/layout/MultiStepLoader";
import { useRegistryStore } from "@/stores/registry-store";
import { useConfigStore } from "@/stores/config-store";
import { useInstalledStore } from "@/stores/installed-store";
import { useProfileStore } from "@/stores/profile-store";
import { useGameStore } from "@/stores/game-store";
import { useTheme } from "@/hooks/use-theme";
import { DownloadNotification } from "@/components/layout/DownloadNotification";
import { HomePage } from "@/pages/HomePage";
import { SearchPage } from "@/pages/SearchPage";
import { ProjectPage } from "@/pages/ProjectPage";
import { SettingsPage } from "@/pages/SettingsPage";
import { LogsPage } from "@/pages/LogsPage";
import { ExtractNotification } from "./components/layout/ExtractNotification";
import { IsStartupReady } from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";

function App() {
  useTheme();
  const [startupReady, setStartupReady] = useState(false);
  const updateInstalledLists = useInstalledStore((s) => s.updateInstalledLists);

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
    EventsOn("registry:update", () => {
      updateInstalledLists();
    });
    let cancelled = false;
    let timer: number | undefined;

    const pollStartupReady = async () => {
      try {
        const ready = await IsStartupReady();
        if (cancelled) return;
        if (ready) {
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
      cancelled = true;
      if (timer !== undefined) {
        window.clearTimeout(timer);
      }
    };
  }, []);

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
    { text: "Starting backend services" },
    { text: "Loading configuration" },
    { text: "Applying theme preferences" },
    { text: "Loading user profile" },
    ...(showRegistrySteps
      ? [
          { text: "Connecting to registry" },
          { text: "Loading installed content" },
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

  const baseLoading = !startupReady || !configInitialized || !profileInitialized;
  const registryLoading =
    showRegistrySteps && (!registryInitialized || !installedInitialized);

  if (baseLoading || registryLoading) {
    return <MultiStepLoader loadingStates={loadingStates} currentStep={currentStep} />;
  }

  // Gate: show setup if not configured OR setup not completed
  if (!isConfigured || !setupCompleted) {
    return (
      <>
        <SetupScreen />
        <Toaster />
      </>
    );
  }

  return (
    <TooltipProvider>
      <Layout>
        <Switch>
          <Route path="/" component={HomePage} />
          <Route path="/search" component={SearchPage} />
          <Route path="/project/:type/:id" component={ProjectPage} />
          <Route path="/logs" component={LogsPage} />
          <Route path="/settings" component={SettingsPage} />
        </Switch>
      </Layout>
      <DownloadNotification />
      <ExtractNotification />
      <Toaster/>
    </TooltipProvider>
  );
}

export default App;
