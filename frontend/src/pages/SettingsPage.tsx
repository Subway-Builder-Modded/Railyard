import { useState } from "react";
import { useConfigStore } from "@/stores/config-store";
import { useProfileStore } from "@/stores/profile-store";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import { FolderOpen, Gamepad2, AlertTriangle } from "lucide-react";

export function SettingsPage() {
  const { config, validation, openDataFolderDialog, openExecutableDialog, saveConfig, clearConfig } = useConfigStore();
  const profile = useProfileStore((s) => s.profile);
  const resetProfile = useProfileStore((s) => s.resetProfile);

  const [confirmAction, setConfirmAction] = useState<"config" | "profile" | null>(null);

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

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>

      {/* Game Paths */}
      <Card>
        <CardHeader>
          <CardTitle>Game Paths</CardTitle>
          <CardDescription>Configure paths to your Metro Maker data and game executable.</CardDescription>
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
              <Badge variant={validation?.metroMakerDataPathValid ? "default" : config?.metroMakerDataPath ? "destructive" : "outline"}>
                {validation?.metroMakerDataPathValid ? "Valid" : config?.metroMakerDataPath ? "Invalid" : "Not Set"}
              </Badge>
              <Button variant="outline" size="sm" onClick={handleChangeDataFolder}>
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
              <Badge variant={validation?.executablePathValid ? "default" : config?.executablePath ? "destructive" : "outline"}>
                {validation?.executablePathValid ? "Valid" : config?.executablePath ? "Invalid" : "Not Set"}
              </Badge>
              <Button variant="outline" size="sm" onClick={handleChangeExecutable}>
                Change
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Preferences */}
      <Card>
        <CardHeader>
          <CardTitle>Preferences</CardTitle>
          <CardDescription>Display and behavior preferences from your profile.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <label className="text-sm font-medium">Theme</label>
            <Select value={profile?.uiPreferences?.theme ?? "system"} disabled>
              <SelectTrigger className="w-[140px]">
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
            <Select value={String(profile?.uiPreferences?.defaultPerPage ?? 12)} disabled>
              <SelectTrigger className="w-[140px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="12">12</SelectItem>
                <SelectItem value="24">24</SelectItem>
                <SelectItem value="48">48</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Danger Zone */}
      <Card className="border-destructive/50">
        <CardHeader>
          <CardTitle className="text-destructive">Danger Zone</CardTitle>
          <CardDescription>Irreversible actions that reset your data.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Reset Configuration</p>
              <p className="text-xs text-muted-foreground">Clear all saved paths and settings.</p>
            </div>
            <Button variant="destructive" size="sm" onClick={() => setConfirmAction("config")}>
              Reset Config
            </Button>
          </div>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Reset Profile</p>
              <p className="text-xs text-muted-foreground">Clear your profile and preferences.</p>
            </div>
            <Button variant="destructive" size="sm" onClick={() => setConfirmAction("profile")}>
              Reset Profile
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Confirmation Dialog */}
      <Dialog open={confirmAction !== null} onOpenChange={(open) => !open && setConfirmAction(null)}>
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
