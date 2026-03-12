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
import { toast } from "sonner";
import {
  ExternalLink,
  MapPin,
  Users,
  Globe,
  Loader2,
  Trash2,
  CheckCircle,
  Download,
} from "lucide-react";
import Markdown from "react-markdown";
import rehypeRaw from "rehype-raw";
import { BrowserOpenURL } from "../../../wailsjs/runtime/runtime";
import { types } from "../../../wailsjs/go/models";
import { formatSourceQuality } from "@/lib/map-filter-values";
import { useDownloadQueueStore } from "@/stores/download-queue-store";

interface ProjectInfoProps {
  type: "mods" | "maps";
  item: types.ModManifest | types.MapManifest;
  latestVersion?: types.VersionInfo;
  latestCompatibleVersion?: types.VersionInfo;
  versionsLoading: boolean;
  gameVersion: string;
}

function isMapManifest(
  item: types.ModManifest | types.MapManifest,
): item is types.MapManifest {
  return "city_code" in item;
}

export function ProjectInfo({
  type,
  item,
  latestVersion,
  latestCompatibleVersion,
  versionsLoading,
  gameVersion,
}: ProjectInfoProps) {
  const [uninstallOpen, setUninstallOpen] = useState(false);
  const [installError, setInstallError] = useState<{
    version: string;
    message: string;
  } | null>(null);
  const [prereleasePrompt, setPrereleasePrompt] = useState(false);
  const { installMod, installMap, getInstalledVersion, isOperating } =
    useInstalledStore();

  const installedVersion = getInstalledVersion(item.id);
  const installing = isOperating(item.id);
  const detailBadges = isMapManifest(item)
    ? [
        item.location,
        formatSourceQuality(item.source_quality),
        item.level_of_detail,
        ...(item.special_demand ?? []),
      ].filter((value): value is string => Boolean(value))
    : (item.tags ?? []);
  // Use the latest compatible version for install/update buttons
  const effectiveVersion = latestCompatibleVersion ?? latestVersion;
  const hasUpdate =
    installedVersion &&
    effectiveVersion &&
    installedVersion !== effectiveVersion.version;
  // No compatible version exists at all
  const noCompatibleVersion =
    gameVersion && latestVersion && !latestCompatibleVersion;

  const handleInstall = async (version: string) => {
    try {
      if (type === "mods") {
        await installMod(item.id, version);
      } else {
        await installMap(item.id, version);
      }
      const { completed, total } = useDownloadQueueStore.getState();
      const queueText = total > 1 ? ` (${completed}/${total} Downloaded)` : "";
      toast.success(
        `${item.name} ${version} installed successfully.${queueText}`,
      );
    } catch (err) {
      setInstallError({
        version,
        message: err instanceof Error ? err.message : String(err),
      });
    }
  };

  const handleInstallClick = (version: string, prerelease?: boolean) => {
    if (prerelease) {
      setPrereleasePrompt(true);
    } else {
      handleInstall(version);
    }
  };

  const renderInstallButton = (v: types.VersionInfo, label: string) => {
    if (noCompatibleVersion) {
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
              No version compatible with your installed game version (
              {gameVersion})
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      );
    }
    return (
      <>
        <Button
          size="sm"
          onClick={() => handleInstallClick(v.version, v.prerelease)}
        >
          <Download className="h-4 w-4 mr-1.5" />
          {label}
        </Button>
        {label.toLowerCase().includes("update") && (
          <Button
            variant="outline"
            size="icon"
            className="h-8 w-8"
            onClick={() => setUninstallOpen(true)}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        )}
      </>
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
          ) : !installedVersion && effectiveVersion ? (
            renderInstallButton(
              effectiveVersion,
              `Install ${effectiveVersion.version}`,
            )
          ) : hasUpdate && effectiveVersion ? (
            renderInstallButton(
              effectiveVersion,
              `Update to ${effectiveVersion.version}`,
            )
          ) : installedVersion ? (
            <>
              <Badge variant="secondary" className="gap-1">
                <CheckCircle className="h-3 w-3" />
                Installed {installedVersion}
              </Badge>
              <Button
                variant="outline"
                size="icon"
                className="h-8 w-8"
                onClick={() => setUninstallOpen(true)}
              >
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
              <span>{item.population.toLocaleString()}</span>
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

      {detailBadges.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {detailBadges.map((badge) => (
            <Badge key={badge} variant="secondary">
              {badge}
            </Badge>
          ))}
        </div>
      )}

      {item.source && (
        <Button
          variant="outline"
          size="sm"
          onClick={() => BrowserOpenURL(item.source!)}
        >
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

      {prereleasePrompt && effectiveVersion && (
        <PrereleaseConfirmDialog
          open={prereleasePrompt}
          onOpenChange={(open) => {
            if (!open) setPrereleasePrompt(false);
          }}
          itemName={item.name}
          version={effectiveVersion.version}
          onConfirm={() => handleInstall(effectiveVersion.version)}
        />
      )}

      {installError && (
        <InstallErrorDialog
          open={!!installError}
          onOpenChange={(open) => {
            if (!open) setInstallError(null);
          }}
          itemName={item.name}
          version={installError.version}
          error={installError.message}
        />
      )}
    </div>
  );
}
