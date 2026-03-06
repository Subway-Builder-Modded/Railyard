import { useState, useEffect, useMemo } from "react";
import { useRoute, Link } from "wouter";
import { useRegistryStore } from "@/stores/registry-store";
import { GetVersions } from "../../wailsjs/go/registry/Registry";
import { types } from "../../wailsjs/go/models";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { ProjectHero } from "@/components/project/ProjectHero";
import { ProjectInfo } from "@/components/project/ProjectInfo";
import { VersionsTable } from "@/components/project/VersionsTable";
import { EmptyState } from "@/components/shared/EmptyState";
import { Separator } from "@/components/ui/separator";
import { CircleAlert } from "lucide-react";

export function ProjectPage() {
  const [, params] = useRoute("/project/:type/:id");
  const mods = useRegistryStore((s) => s.mods);
  const maps = useRegistryStore((s) => s.maps);

  const type = params?.type as "mods" | "maps" | undefined;
  const id = params?.id;

  const item =
    type === "mods"
      ? mods.find((m) => m.id === id)
      : type === "maps"
        ? maps.find((m) => m.id === id)
        : undefined;

  const [versions, setVersions] = useState<types.VersionInfo[]>([]);
  const [versionsLoading, setVersionsLoading] = useState(true);
  const [versionsError, setVersionsError] = useState<string | null>(null);

  useEffect(() => {
    if (!item) return;
    const source = item.update.type === "github" ? item.update.repo : item.update.url;
    if (!source) {
      setVersionsLoading(false);
      setVersionsError("No update source configured");
      return;
    }
    let cancelled = false;
    setVersionsLoading(true);
    setVersionsError(null);
    GetVersions(item.update.type, source)
      .then((v) => {
        if (!cancelled) {
          setVersions(v || []);
          setVersionsLoading(false);
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setVersionsError(err instanceof Error ? err.message : String(err));
          setVersionsLoading(false);
        }
      });
    return () => { cancelled = true; };
  }, [item?.update.type, item?.update.repo, item?.update.url]);

  const latestVersion = versions[0];
  const gallery = useMemo(() => item?.gallery || [], [item?.gallery]);

  if (!item || !type) {
    return (
      <EmptyState
        icon={CircleAlert}
        title="Project not found"
        description="The mod or map you're looking for doesn't exist in the registry."
      />
    );
  }

  return (
    <div className="space-y-6">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link href="/">Home</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link href="/search">Browse</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>{item.name}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <ProjectHero type={type} id={item.id} gallery={gallery} />

      <ProjectInfo type={type} item={item} latestVersion={latestVersion} versionsLoading={versionsLoading} />

      <Separator />

      <VersionsTable type={type} itemId={item.id} update={item.update} versions={versions} loading={versionsLoading} error={versionsError} />
    </div>
  );
}
