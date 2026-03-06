import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { useInstalledStore } from "@/stores/installed-store";
import { useProfileStore } from "@/stores/profile-store";
import { UninstallDialog } from "@/components/dialogs/UninstallDialog";
import { toast } from "sonner";
import { ExternalLink, MapPin, Users, Globe, Bell, BellOff, Loader2, Trash2, CheckCircle, Download } from "lucide-react";
import { BrowserOpenURL } from "../../../wailsjs/runtime/runtime";
import { types } from "../../../wailsjs/go/models";

interface ProjectInfoProps {
  type: "mods" | "maps";
  item: types.ModManifest | types.MapManifest;
  latestVersion?: types.VersionInfo;
  versionsLoading: boolean;
}

function isMapManifest(
  item: types.ModManifest | types.MapManifest
): item is types.MapManifest {
  return "city_code" in item;
}

export function ProjectInfo({ type, item, latestVersion, versionsLoading }: ProjectInfoProps) {
  const [uninstallOpen, setUninstallOpen] = useState(false);
  const { installMod, installMap, getInstalledVersion, isOperating } = useInstalledStore();
  const { isSubscribed, updateSubscription } = useProfileStore();

  const installedVersion = getInstalledVersion(item.id);
  const installing = isOperating(item.id);
  const subscribed = isSubscribed(type, item.id);
  const hasUpdate = installedVersion && latestVersion && installedVersion !== latestVersion.version;

  const handleInstall = async (version: string) => {
    try {
      if (type === "mods") {
        await installMod(item.id, version);
      } else {
        await installMap(item.id, version);
      }
      toast.success(`${item.name} ${version} installed successfully.`);
    } catch {
      toast.error(`Failed to install ${item.name}.`);
    }
  };

  const handleSubscribe = async () => {
    try {
      const action = subscribed ? "unsubscribe" : "subscribe";
      const version = latestVersion?.version || installedVersion || "";
      await updateSubscription(type, item.id, action, version);
      toast.success(subscribed ? `Unsubscribed from ${item.name}.` : `Subscribed to ${item.name}.`);
    } catch {
      toast.error("Failed to update subscription.");
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">{item.name}</h1>
          <p className="text-muted-foreground mt-1">by {item.author}</p>
        </div>

        <div className="flex items-center gap-2 flex-shrink-0">
          <Button variant="outline" size="sm" onClick={handleSubscribe}>
            {subscribed ? <BellOff className="h-4 w-4 mr-1.5" /> : <Bell className="h-4 w-4 mr-1.5" />}
            {subscribed ? "Unsubscribe" : "Subscribe"}
          </Button>

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
            <Button size="sm" onClick={() => handleInstall(latestVersion.version)}>
              <Download className="h-4 w-4 mr-1.5" />
              Install {latestVersion.version}
            </Button>
          ) : hasUpdate && latestVersion ? (
            <Button size="sm" onClick={() => handleInstall(latestVersion.version)}>
              <Download className="h-4 w-4 mr-1.5" />
              Update to {latestVersion.version}
            </Button>
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

      <p className="text-sm leading-relaxed">{item.description}</p>

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
    </div>
  );
}
