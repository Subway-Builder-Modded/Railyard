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
import { Download, FileText, ArrowDownToLine, Loader2, CheckCircle } from "lucide-react";
import { useInstalledStore } from "@/stores/installed-store";
import { types } from "../../../wailsjs/go/models";
import { EmptyState } from "@/components/shared/EmptyState";
import { ErrorBanner } from "@/components/shared/ErrorBanner";
import { toast } from "sonner";

interface VersionsTableProps {
  type: "mods" | "maps";
  itemId: string;
  update: types.UpdateConfig;
  versions: types.VersionInfo[];
  loading: boolean;
  error: string | null;
}

export function VersionsTable({ type, itemId, update, versions, loading, error }: VersionsTableProps) {
  const { getInstalledVersion, installMod, installMap, isOperating } = useInstalledStore();
  const installedVersion = getInstalledVersion(itemId);

  const handleInstall = async (version: string) => {
    try {
      if (type === "mods") {
        await installMod(itemId, version);
      } else {
        await installMap(itemId, version);
      }
      toast.success(`Installed ${version} successfully.`);
    } catch {
      toast.error(`Failed to install ${version}.`);
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

  return (
    <div className="space-y-3">
      <h2 className="text-xl font-semibold">Versions</h2>
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Version</TableHead>
              <TableHead>Date</TableHead>
              {update.type === "custom" && <TableHead>Game Version</TableHead>}
              <TableHead>Changelog</TableHead>
              <TableHead>Downloads</TableHead>
              <TableHead className="w-[100px]"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {versions.map((v) => {
              const isThisInstalled = installedVersion === v.version;
              const isInstalling = isOperating(itemId);

              return (
                <TableRow key={v.version}>
                  <TableCell className="font-mono font-medium">
                    {v.version}
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {formatDate(v.date)}
                  </TableCell>
                  {update.type === "custom" && (
                    <TableCell className="text-muted-foreground font-mono text-xs">
                      {v.game_version}
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
                    ) : isInstalling ? (
                      <Button variant="outline" size="sm" disabled>
                        <Loader2 className="h-4 w-4 animate-spin" />
                      </Button>
                    ) : (
                      <Button variant="outline" size="sm" onClick={() => handleInstall(v.version)}>
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
    </div>
  );
}
