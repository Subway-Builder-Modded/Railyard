import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useInstalledStore } from "@/stores/installed-store";
import { UninstallDialog } from "@/components/dialogs/UninstallDialog";
import { InstallErrorDialog } from "@/components/dialogs/InstallErrorDialog";
import { PrereleaseConfirmDialog } from "@/components/dialogs/PrereleaseConfirmDialog";
import { isCompatible } from "@/lib/semver";
import { toast } from "sonner";
import { ExternalLink, MapPin, Users, Globe, Loader2, Trash2, CheckCircle, Download } from "lucide-react";
import Markdown from "react-markdown";
import rehypeRaw from "rehype-raw";
import { BrowserOpenURL } from "../../../wailsjs/runtime/runtime";
import { types } from "../../../wailsjs/go/models";

interface ProjectInfoProps {
  type: "mods" | "maps";
  item: types.ModManifest | types.MapManifest;
  latestVersion?: types.VersionInfo;
  versionsLoading: boolean;
  gameVersion: string;
}

function isMapManifest(
  item: types.ModManifest | types.MapManifest
): item is types.MapManifest {
  return "city_code" in item;
}

export function ProjectInfo({ type, item, latestVersion, versionsLoading, gameVersion }: ProjectInfoProps) {
  const [uninstallOpen, setUninstallOpen] = useState(false);
  const [installError, setInstallError] = useState<{ version: string; message: string } | null>(null);
  const [prereleasePrompt, setPrereleasePrompt] = useState(false);
  const { installMod, installMap, getInstalledVersion, isOperating } = useInstalledStore();

  const installedVersion = getInstalledVersion(item.id);
  const installing = isOperating(item.id);
  const hasUpdate = installedVersion && latestVersion && installedVersion !== latestVersion.version;

  const latestCompat = latestVersion
    ? isCompatible(gameVersion, latestVersion.game_version)
    : null;
  const latestIncompatible = latestCompat === false;

  const handleInstall = async (version: string) => {
    try {
      if (type === "mods") {
        await installMod(item.id, version);
      } else {
        await installMap(item.id, version);
      }
      toast.success(`${item.name} ${version} installed successfully.`);
    } catch (err) {
      setInstallError({ version, message: err instanceof Error ? err.message : String(err) });
    }
  };

  const handleInstallClick = (version: string) => {
    if (latestVersion?.prerelease) {
      setPrereleasePrompt(true);
    } else {
      handleInstall(version);
    }
  };

  const renderInstallButton = (version: string, label: string) => {
    if (latestIncompatible) {
      return (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <span>
                <Button size="sm" disabled>
                  <Download className="h-4 w-4 mr-1.5" />
                  {label}
                </Button>
              </span>
            </TooltipTrigger>
            <TooltipContent>
              Not compatible with your installed game version
              (you have {gameVersion}, need {latestVersion?.game_version})
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      );
    }
    return (
      <Button size="sm" onClick={() => handleInstallClick(version)}>
        <Download className="h-4 w-4 mr-1.5" />
        {label}
      </Button>
    );
  };

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{item.name}</h1>
          <p className="text-muted-foreground mt-1">by {item.author}</p>
        </div>

        <div className="flex items-center gap-2 flex-shrink-0">
          {versionsLoading ? (
            <Button size="sm" disabled>
              <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
              Loading...
            </Button>
          ) : installing ? (
            <Button size="sm" disabled>
              <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
              Installing...
            </Button>
          ) : !installedVersion && latestVersion ? (
            renderInstallButton(latestVersion.version, `Install ${latestVersion.version}`)
          ) : hasUpdate && latestVersion ? (
            renderInstallButton(latestVersion.version, `Update to ${latestVersion.version}`)
          ) : installedVersion ? (
            <>
              <Badge variant="secondary" className="gap-1">
                <CheckCircle className="h-3 w-3" />
                Installed {installedVersion}
              </Badge>
              <Button variant="outline" size="icon" className="h-8 w-8" onClick={() => setUninstallOpen(true)}>
                <Trash2 className="h-4 w-4" />
              </Button>
            </>
          ) : null}
        </div>
      </div>

      {isMapManifest(item) && (
        <div className="flex items-center gap-4 text-sm">
          {item.city_code && (
            <div className="flex items-center gap-1.5">
              <MapPin className="h-4 w-4 text-muted-foreground" />
              <span className="font-mono font-bold">{item.city_code}</span>
              {item.country && (
                <span className="text-muted-foreground">{item.country}</span>
              )}
            </div>
          )}
          {item.population > 0 && (
            <div className="flex items-center gap-1.5">
              <Users className="h-4 w-4 text-muted-foreground" />
              <span>Pop. {item.population.toLocaleString()}</span>
            </div>
          )}
        </div>
      )}

      <Separator />

      <div className="text-sm leading-relaxed prose prose-sm prose-neutral dark:prose-invert max-w-none">
        <Markdown
          rehypePlugins={[rehypeRaw]}
          components={{
            a: ({ href, children, ...props }) => (
              <a
                {...props}
                href={href}
                onClick={(e) => {
                  if (href) {
                    e.preventDefault();
                    BrowserOpenURL(href);
                  }
                }}
              >
                {children}
              </a>
            ),
          }}
        >
          {item.description}
        </Markdown>
      </div>

      {item.tags && item.tags.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {item.tags.map((tag) => (
            <Badge key={tag} variant="secondary">
              {tag}
            </Badge>
          ))}
        </div>
      )}

      {item.source && (
        <Button variant="outline" size="sm" onClick={() => BrowserOpenURL(item.source!)}>
          <Globe className="h-4 w-4 mr-1.5" />
          View Source
          <ExternalLink className="h-3 w-3 ml-1.5" />
        </Button>
      )}

      <UninstallDialog
        open={uninstallOpen}
        onOpenChange={setUninstallOpen}
        type={type}
        id={item.id}
        name={item.name}
      />

      {prereleasePrompt && latestVersion && (
        <PrereleaseConfirmDialog
          open={prereleasePrompt}
          onOpenChange={(open) => { if (!open) setPrereleasePrompt(false); }}
          itemName={item.name}
          version={latestVersion.version}
          onConfirm={() => handleInstall(latestVersion.version)}
        />
      )}

      {installError && (
        <InstallErrorDialog
          open={!!installError}
          onOpenChange={(open) => { if (!open) setInstallError(null); }}
          itemName={item.name}
          version={installError.version}
          error={installError.message}
        />
      )}
    </div>
  );
}
