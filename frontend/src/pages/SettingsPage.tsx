import { useState } from "react";
import { useConfigStore } from "@/stores/config-store";
import { useProfileStore } from "@/stores/profile-store";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { toast } from "sonner";
import {
  FolderOpen,
  Gamepad2,
  Github,
  AlertTriangle,
  RefreshCw,
} from "lucide-react";
import { ManuallyCheckForUpdates } from "../../wailsjs/go/main/App";

const PAGE_SIZE_OPTIONS = [12, 24, 48] as const;
const THEME_OPTIONS = ["dark", "light", "system"] as const;

export function SettingsPage() {
  const {
    config,
    validation,
    hasGithubToken,
    openDataFolderDialog,
    openExecutableDialog,
    saveConfig,
    clearConfig,
    updateGithubToken,
    clearGithubToken,
    updateCheckForUpdatesOnLaunch,
  } = useConfigStore();
  const profile = useProfileStore((s) => s.profile);
  const resetProfile = useProfileStore((s) => s.resetProfile);
  const updateUIPreferences = useProfileStore((s) => s.updateUIPreferences);

  const [confirmAction, setConfirmAction] = useState<
    "config" | "profile" | null
  >(null);
  const [githubTokenDialogOpen, setGithubTokenDialogOpen] = useState(false);
  const [githubTokenDraft, setGithubTokenDraft] = useState("");

  const handleUpdatesCheck = async () => {
    try {
      await ManuallyCheckForUpdates();
      toast.success("No updates found, or installation was cancelled.");
    } catch (error: any) {
      toast.error("Failed to check for updates.");
    }
  };

  const handleChangeUpdatesOnLaunch = async () => {
    try {
      const newValue = !config?.checkForUpdatesOnLaunch;
      await updateCheckForUpdatesOnLaunch(newValue);
      toast.success(
        `Check for updates on launch ${newValue ? "enabled" : "disabled"}.`,
      );
    } catch {
      toast.error("Failed to update check for updates on launch setting.");
    }
  };

  const handleThemeChange = async (theme: string) => {
    if (
      !profile ||
      !THEME_OPTIONS.includes(theme as (typeof THEME_OPTIONS)[number])
    )
      return;

    try {
      await updateUIPreferences(
        theme,
        profile.uiPreferences?.defaultPerPage ?? 12,
      );
      toast.success("Theme updated.");
    } catch {
      toast.error("Failed to update theme.");
    }
  };

  const handleDefaultPerPageChange = async (value: string) => {
    if (!profile) return;
    const parsed = Number.parseInt(value, 10);

    if (
      !PAGE_SIZE_OPTIONS.includes(parsed as (typeof PAGE_SIZE_OPTIONS)[number])
    )
      return;

    try {
      await updateUIPreferences(profile.uiPreferences?.theme ?? "dark", parsed);
      toast.success("Default cards per page updated.");
    } catch {
      toast.error("Failed to update default cards per page.");
    }
  };

  const handleChangeDataFolder = async () => {
    try {
      const result = await openDataFolderDialog(false);
      if (result.source === "cancelled") return;
      await saveConfig();
      toast.success("Data folder path updated.");
    } catch {
      toast.error("Failed to update data folder path.");
    }
  };

  const handleChangeExecutable = async () => {
    try {
      const result = await openExecutableDialog(false);
      if (result.source === "cancelled") return;
      await saveConfig();
      toast.success("Executable path updated.");
    } catch {
      toast.error("Failed to update executable path.");
    }
  };

  const handleConfirm = async () => {
    try {
      if (confirmAction === "config") {
        await clearConfig();
        toast.success("Configuration has been reset.");
      } else if (confirmAction === "profile") {
        await resetProfile();
        toast.success("Profile has been reset.");
      }
    } catch {
      toast.error(`Failed to reset ${confirmAction}.`);
    } finally {
      setConfirmAction(null);
    }
  };

  const handleSaveGithubToken = async () => {
    try {
      await updateGithubToken(githubTokenDraft);
      await saveConfig();
      setGithubTokenDraft("");
      setGithubTokenDialogOpen(false);
      toast.success("GitHub token updated.");
    } catch {
      toast.error("Failed to update GitHub token.");
    }
  };

  const handleClearGithubToken = async () => {
    try {
      await clearGithubToken();
      await saveConfig();
      setGithubTokenDraft("");
      setGithubTokenDialogOpen(false);
      toast.success("GitHub token cleared.");
    } catch {
      toast.error("Failed to clear GitHub token.");
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>

      <Card>
        <CardHeader>
          <CardTitle>Global Settings</CardTitle>
          <CardDescription>
            Configure system-wide behavior and defaults.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3 min-w-0">
              <FolderOpen className="h-5 w-5 shrink-0 text-muted-foreground" />
              <div className="min-w-0">
                <p className="text-sm font-medium">Data Folder</p>
                <p className="text-xs text-muted-foreground font-mono truncate">
                  {config?.metroMakerDataPath || "Not set"}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Badge
                variant={
                  validation?.metroMakerDataPathValid
                    ? "default"
                    : config?.metroMakerDataPath
                      ? "destructive"
                      : "outline"
                }
              >
                {validation?.metroMakerDataPathValid
                  ? "Valid"
                  : config?.metroMakerDataPath
                    ? "Invalid"
                    : "Not Set"}
              </Badge>
              <Button
                variant="outline"
                size="sm"
                onClick={handleChangeDataFolder}
              >
                Change
              </Button>
            </div>
          </div>

          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3 min-w-0">
              <Gamepad2 className="h-5 w-5 shrink-0 text-muted-foreground" />
              <div className="min-w-0">
                <p className="text-sm font-medium">Game Executable</p>
                <p className="text-xs text-muted-foreground font-mono truncate">
                  {config?.executablePath || "Not set"}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Badge
                variant={
                  validation?.executablePathValid
                    ? "default"
                    : config?.executablePath
                      ? "destructive"
                      : "outline"
                }
              >
                {validation?.executablePathValid
                  ? "Valid"
                  : config?.executablePath
                    ? "Invalid"
                    : "Not Set"}
              </Badge>
              <Button
                variant="outline"
                size="sm"
                onClick={handleChangeExecutable}
              >
                Change
              </Button>
            </div>
          </div>

          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3 min-w-0">
              <Github className="h-5 w-5 shrink-0 text-muted-foreground" />
              <div className="min-w-0">
                <p className="text-sm font-medium">GitHub Token (Optional)</p>
                <p className="text-xs text-muted-foreground font-mono truncate">
                  {hasGithubToken ? "********" : "Not set"}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Badge variant={hasGithubToken ? "default" : "outline"}>
                {hasGithubToken ? "Set" : "Unset"}
              </Badge>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setGithubTokenDialogOpen(true)}
              >
                Change
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Preferences</CardTitle>
          <CardDescription>
            Display and behavior preferences from your profile.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <label className="text-sm font-medium">Theme</label>
            <Select
              value={profile?.uiPreferences?.theme ?? "system"}
              onValueChange={handleThemeChange}
            >
              <SelectTrigger className="w-35">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="dark">Dark</SelectItem>
                <SelectItem value="light">Light</SelectItem>
                <SelectItem value="system">System</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="flex items-center justify-between">
            <label className="text-sm font-medium">Default Per Page</label>
            <Select
              value={String(profile?.uiPreferences?.defaultPerPage ?? 12)}
              onValueChange={handleDefaultPerPageChange}
            >
              <SelectTrigger className="w-35">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="12">12</SelectItem>
                <SelectItem value="24">24</SelectItem>
                <SelectItem value="48">48</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="flex items-center justify-between">
            <label className="text-sm font-medium">
              Check For Updates On Launch
            </label>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={handleChangeUpdatesOnLaunch}
              >
                {config?.checkForUpdatesOnLaunch ? "Disable" : "Enable"}
              </Button>
              <Button variant="outline" size="sm" onClick={handleUpdatesCheck}>
                <RefreshCw />
                Check For Updates
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Dialog
        open={githubTokenDialogOpen}
        onOpenChange={(open) => {
          setGithubTokenDialogOpen(open);
          if (!open) {
            setGithubTokenDraft("");
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit GitHub Token</DialogTitle>
            <DialogDescription>
              Provide a GitHub token to avoid unauthorized Github API rate
              limits.
            </DialogDescription>
          </DialogHeader>
          <Input
            type="password"
            placeholder={hasGithubToken ? "********" : "github_pat_..."}
            value={githubTokenDraft}
            onChange={(event) => setGithubTokenDraft(event.target.value)}
            className="font-mono"
          />
          <DialogFooter className="gap-2">
            <Button
              variant="outline"
              onClick={handleClearGithubToken}
              disabled={!hasGithubToken}
            >
              Clear
            </Button>
            <Button
              onClick={handleSaveGithubToken}
              disabled={githubTokenDraft.trim() === ""}
            >
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Card className="border-destructive/50">
        <CardHeader>
          <CardTitle className="text-destructive">Danger Zone</CardTitle>
          <CardDescription>
            Irreversible actions that reset your data.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Reset Configuration</p>
              <p className="text-xs text-muted-foreground">
                Clear all saved paths and settings.
              </p>
            </div>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setConfirmAction("config")}
            >
              Reset Config
            </Button>
          </div>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Reset Profile</p>
              <p className="text-xs text-muted-foreground">
                Clear your profile and preferences.
              </p>
            </div>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setConfirmAction("profile")}
            >
              Reset Profile
            </Button>
          </div>
        </CardContent>
      </Card>

      <Dialog
        open={confirmAction !== null}
        onOpenChange={(open) => !open && setConfirmAction(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              Are you sure?
            </DialogTitle>
            <DialogDescription>
              {confirmAction === "config"
                ? "This will reset all configuration including game paths. You will need to set them up again."
                : "This will reset your profile and all preferences to defaults."}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmAction(null)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleConfirm}>
              Reset
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
