import { useState } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Download, FileText, ArrowDownToLine, Loader2, CheckCircle } from "lucide-react";
import { useInstalledStore } from "@/stores/installed-store";
import { types } from "../../../wailsjs/go/models";
import { EmptyState } from "@/components/shared/EmptyState";
import { ErrorBanner } from "@/components/shared/ErrorBanner";
import { InstallErrorDialog } from "@/components/dialogs/InstallErrorDialog";
import { PrereleaseConfirmDialog } from "@/components/dialogs/PrereleaseConfirmDialog";
import { isCompatible } from "@/lib/semver";
import { toast } from "sonner";

interface VersionsTableProps {
  type: "mods" | "maps";
  itemId: string;
  itemName: string;
  update: types.UpdateConfig;
  versions: types.VersionInfo[];
  loading: boolean;
  error: string | null;
  gameVersion: string;
}

export function VersionsTable({ type, itemId, itemName, versions, loading, error, gameVersion }: VersionsTableProps) {
  const { getInstalledVersion, installMod, installMap, isOperating } = useInstalledStore();
  const installedVersion = getInstalledVersion(itemId);
  const [installError, setInstallError] = useState<{ version: string; message: string } | null>(null);
  const [prereleasePrompt, setPrereleasePrompt] = useState<{ version: string } | null>(null);

  const doInstall = async (version: string) => {
    try {
      if (type === "mods") {
        await installMod(itemId, version);
      } else {
        await installMap(itemId, version);
      }
      toast.success(`Installed ${version} successfully.`);
    } catch (err) {
      setInstallError({ version, message: err instanceof Error ? err.message : String(err) });
    }
  };

  const handleInstall = (version: string, prerelease: boolean) => {
    if (prerelease) {
      setPrereleasePrompt({ version });
    } else {
      doInstall(version);
    }
  };

  if (loading) {
    return (
      <div className="space-y-3">
        <h2 className="text-xl font-semibold">Versions</h2>
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full" />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-3">
        <h2 className="text-xl font-semibold">Versions</h2>
        <ErrorBanner message={error} />
      </div>
    );
  }

  if (versions.length === 0) {
    return (
      <div className="space-y-3">
        <h2 className="text-xl font-semibold">Versions</h2>
        <EmptyState icon={FileText} title="No versions available" />
      </div>
    );
  }

  const formatDate = (dateStr: string) => {
    try {
      return new Date(dateStr).toLocaleDateString(undefined, {
        year: "numeric",
        month: "short",
        day: "numeric",
      });
    } catch {
      return dateStr;
    }
  };

  const hasAnyGameVersion = versions.some((v) => v.game_version);

  return (
    <div className="space-y-3">
      <h2 className="text-xl font-semibold">Versions</h2>
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Version</TableHead>
              <TableHead>Date</TableHead>
              {hasAnyGameVersion && <TableHead>Game Version</TableHead>}
              <TableHead>Changelog</TableHead>
              <TableHead>Downloads</TableHead>
              <TableHead className="w-[100px]"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {versions.map((v) => {
              const isThisInstalled = installedVersion === v.version;
              const isInstalling = isOperating(itemId);
              const compat = isCompatible(gameVersion, v.game_version);
              const incompatible = compat === false;

              return (
                <TableRow key={v.version} className={incompatible ? "opacity-50" : ""}>
                  <TableCell className="font-mono font-medium">
                    <span className="flex items-center gap-2">
                      {v.version}
                      {v.prerelease && (
                        <Badge variant="outline" className="text-yellow-600 border-yellow-600">
                          Beta
                        </Badge>
                      )}
                    </span>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {formatDate(v.date)}
                  </TableCell>
                  {hasAnyGameVersion && (
                    <TableCell className="text-muted-foreground font-mono text-xs">
                      {v.game_version || "\u2014"}
                    </TableCell>
                  )}
                  <TableCell className="text-sm text-muted-foreground max-w-xs truncate">
                    {v.changelog}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    <div className="flex items-center gap-1.5">
                      <ArrowDownToLine className="h-3 w-3" />
                      {v.downloads.toLocaleString()}
                    </div>
                  </TableCell>
                  <TableCell>
                    {isThisInstalled ? (
                      <Badge variant="secondary" className="gap-1">
                        <CheckCircle className="h-3 w-3" />
                        Installed
                      </Badge>
                    ) : incompatible ? (
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span>
                              <Button variant="outline" size="sm" disabled>
                                <Download className="h-4 w-4" />
                              </Button>
                            </span>
                          </TooltipTrigger>
                          <TooltipContent>
                            Not compatible with your installed game version
                            (you have {gameVersion}, need {v.game_version})
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    ) : isInstalling ? (
                      <Button variant="outline" size="sm" disabled>
                        <Loader2 className="h-4 w-4 animate-spin" />
                      </Button>
                    ) : (
                      <Button variant="outline" size="sm" onClick={() => handleInstall(v.version, v.prerelease)}>
                        <Download className="h-4 w-4" />
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>

      {prereleasePrompt && (
        <PrereleaseConfirmDialog
          open={!!prereleasePrompt}
          onOpenChange={(open) => { if (!open) setPrereleasePrompt(null); }}
          itemName={itemName}
          version={prereleasePrompt.version}
          onConfirm={() => doInstall(prereleasePrompt.version)}
        />
      )}

      {installError && (
        <InstallErrorDialog
          open={!!installError}
          onOpenChange={(open) => { if (!open) setInstallError(null); }}
          itemName={itemName}
          version={installError.version}
          error={installError.message}
        />
      )}
    </div>
  );
}
