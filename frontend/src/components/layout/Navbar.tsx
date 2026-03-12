import { useState } from "react";
import { Link, useLocation } from "wouter";
import { Play, Square, RefreshCw, TrainTrack } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { useRegistryStore } from "@/stores/registry-store";
import { useConfigStore } from "@/stores/config-store";
import { useInstalledStore } from "@/stores/installed-store";
import { useGameStore } from "@/stores/game-store";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

const navLinks = [
  { href: "/", label: "Home" },
  { href: "/library", label: "Library" },
  { href: "/search", label: "Browse" },
  { href: "/logs", label: "Logs" },
  { href: "/settings", label: "Settings" },
] as const;

const MOD_REMINDER_KEY = "railyard:mod-reminder-acknowledged";

export function Navbar() {
  const [location] = useLocation();
  const { refresh, loading, refreshing } = useRegistryStore();
  const canLaunch = useConfigStore((s) => s.validation?.executablePathValid);
  const { running, launch, stop } = useGameStore();
  const installedMaps = useInstalledStore((s) => s.installedMaps);
  const [showModReminder, setShowModReminder] = useState(false);

  const handleLaunch = async () => {
    const hasMaps = installedMaps.length > 0;
    const alreadyAcknowledged = localStorage.getItem(MOD_REMINDER_KEY) === "true";

    if (hasMaps && !alreadyAcknowledged) {
      setShowModReminder(true);
      return;
    }

    try {
      await launch();
    } catch (err) {
      toast.error(String(err) || "Failed to launch game.");
    }
  };

  const handleAcknowledgeAndLaunch = async () => {
    localStorage.setItem(MOD_REMINDER_KEY, "true");
    setShowModReminder(false);
    try {
      await launch();
    } catch (err) {
      toast.error(String(err) || "Failed to launch game.");
    }
  };

  const handleStop = async () => {
    try {
      await stop();
    } catch (err) {
      toast.error(String(err) || "Failed to stop game.");
    }
  };

  return (
    <header className="sticky top-0 z-50 border-b bg-background/95 backdrop-blur">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex h-14 items-center justify-between">
        <div className="flex items-center gap-6">
          <Link href="/" className="flex items-center gap-2 font-bold text-lg">
            <TrainTrack className="h-5 w-5" />
            Railyard
          </Link>
          <nav className="flex items-center gap-4">
            {navLinks.map(({ href, label }) => (
              <Link
                key={href}
                href={href}
                className={cn(
                  "text-sm transition-colors hover:text-foreground flex items-center gap-1.5",
                  location === href
                    ? "text-foreground font-medium"
                    : "text-muted-foreground"
                )}
              >
                {label}
              </Link>
            ))}
          </nav>
        </div>
        <div className="flex items-center gap-1">
          {running ? (
            <Button
              variant="ghost"
              size="sm"
              onClick={handleStop}
              className="text-destructive hover:text-destructive"
            >
              <Square className="h-4 w-4 mr-1.5" />
              Running
            </Button>
          ) : (
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleLaunch}
                    disabled={!canLaunch}
                  >
                    <Play className="h-4 w-4 mr-1.5" />
                    Launch
                  </Button>
                </span>
              </TooltipTrigger>
              {!canLaunch && (
                <TooltipContent>
                  Configure game executable in Settings first
                </TooltipContent>
              )}
            </Tooltip>
          )}
          <Button variant="ghost" size="icon" onClick={refresh} disabled={loading || refreshing}>
            <RefreshCw className={cn("h-4 w-4", (loading || refreshing) && "animate-spin")} />
          </Button>
        </div>
      </div>

      <Dialog open={showModReminder} onOpenChange={setShowModReminder}>
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>Enable Railyard Map Loader</DialogTitle>
            <DialogDescription>
              You have custom maps installed. To use them in-game, make sure the{" "}
              <span className="font-semibold text-foreground">Railyard Map Loader</span>{" "}
              mod is enabled in the game's mod manager.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowModReminder(false)}>
              Cancel
            </Button>
            <Button onClick={handleAcknowledgeAndLaunch}>
              Got it, launch game
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </header>
  );
}
