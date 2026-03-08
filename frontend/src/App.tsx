import { useEffect } from "react";
import { Route, Switch } from "wouter";
import { TooltipProvider } from "@/components/ui/tooltip";
import { Toaster } from "@/components/ui/sonner";
import { Layout } from "@/components/layout/Layout";
import { SetupScreen } from "@/components/setup/SetupScreen";
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

function App() {
  useTheme();

  const initConfig = useConfigStore((s) => s.initialize);
  const configInitialized = useConfigStore((s) => s.initialized);
  const isConfigured = useConfigStore((s) => s.validation?.isConfigured ?? false);

  const initProfile = useProfileStore((s) => s.initialize);

  const initRegistry = useRegistryStore((s) => s.initialize);
  const initInstalled = useInstalledStore((s) => s.initialize);
  const initGame = useGameStore((s) => s.initialize);

  // Phase 1: config + profile + game events
  useEffect(() => {
    initConfig();
    initProfile();
    initGame();
  }, [initConfig, initProfile, initGame]);

  // Phase 2: registry + installed (only when configured)
  useEffect(() => {
    if (configInitialized && isConfigured) {
      initRegistry();
      initInstalled();
    }
  }, [configInitialized, isConfigured, initRegistry, initInstalled]);

  // Gate: show loading until config is ready
  if (!configInitialized) {
    return null;
  }

  // Gate: show setup if not configured
  if (!isConfigured) {
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
      <Toaster />
    </TooltipProvider>
  );
}

export default App;
