import { ExternalLink, Loader2, Package } from 'lucide-react';
import { useMemo } from 'react';
import { Link } from 'wouter';

import { EmptyState } from '@/components/shared/EmptyState';
import { GalleryImage } from '@/components/shared/GalleryImage';
import { SubwayBuilderIcon } from '@/components/shared/SubwayBuilderIcon';
import { Badge } from '@/components/ui/badge';
import type { AssetType } from '@/lib/asset-types';
import { assetTypeToListingPath } from '@/lib/asset-types';
import { useRegistryStore } from '@/stores/registry-store';

import type { types } from '../../../wailsjs/go/models';

interface ChangelogDependenciesProps {
  type: AssetType;
  itemId: string;
  versionInfo: types.VersionInfo;
  resolvedDeps: Record<string, types.DependencyListEntry> | null;
  resolving: boolean;
}

interface FlatDep {
  id: string;
  range: string;
  kind: 'direct' | 'indirect';
}

export function ChangelogDependencies({
  type,
  itemId: _itemId,
  versionInfo,
  resolvedDeps,
  resolving,
}: ChangelogDependenciesProps) {
  const mods = useRegistryStore((s) => s.mods);

  const rawDeps = versionInfo.dependencies ?? {};
  const gameDep = versionInfo.game_version;
  const directIds = useMemo(() => new Set(Object.keys(rawDeps)), [rawDeps]);

  const flatDeps = useMemo<FlatDep[]>(() => {
    const result: FlatDep[] = [];
    if (resolvedDeps !== null) {
      for (const id of directIds) {
        result.push({ id, range: rawDeps[id], kind: 'direct' });
      }
      for (const [id, entry] of Object.entries(resolvedDeps)) {
        if (!directIds.has(id)) {
          result.push({
            id,
            range: entry.installCandidate?.version ?? '?',
            kind: 'indirect',
          });
        }
      }
    } else if (type === 'mod') {
      for (const id of directIds) {
        result.push({ id, range: rawDeps[id], kind: 'direct' });
      }
    }
    return result;
  }, [resolvedDeps, directIds, rawDeps, type]);

  if (!gameDep && flatDeps.length === 0 && !resolving) {
    return (
      <EmptyState
        icon={Package}
        title="No dependencies"
        description="This version has no dependencies."
      />
    );
  }

  return (
    <div className="space-y-3">
      {gameDep && (
        <div className="flex items-center justify-between gap-3 rounded-xl border border-primary/25 bg-primary/5 px-4 py-3.5">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-primary/25 bg-primary/10">
              <SubwayBuilderIcon className="h-5 w-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <p className="text-sm font-semibold">Subway Builder</p>
                <Badge variant="secondary" size="sm">
                  Direct
                </Badge>
              </div>
              <p className="text-xs text-muted-foreground">Base Game</p>
            </div>
          </div>
          <Badge variant="secondary" className="font-mono">
            {gameDep}
          </Badge>
        </div>
      )}

      {(flatDeps.length > 0 || resolving) && (
        <div className="overflow-hidden rounded-xl border border-border bg-card">
          <div className="border-b border-border px-4 py-3">
            <h3 className="text-sm font-semibold">Mod Dependencies</h3>
          </div>
          {resolving ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : (
            <div className="divide-y divide-border/50">
              {flatDeps.map(({ id, range, kind }) => {
                const mod = mods.find((m) => m.id === id);
                const name = mod?.name ?? id;

                return (
                  <Link
                    key={id}
                    href={`/project/${assetTypeToListingPath('mod')}/${id}`}
                    className="flex items-center justify-between gap-3 px-4 py-3 transition-colors hover:bg-accent/30"
                  >
                    <div className="flex min-w-0 items-center gap-2.5">
                      <div className="relative h-8 w-8 shrink-0 overflow-hidden rounded-md">
                        <GalleryImage
                          type="mod"
                          id={id}
                          imagePath={mod?.gallery?.[0]}
                          className="absolute inset-0 h-full w-full object-cover"
                          fallbackIconClassName="h-3.5 w-3.5"
                        />
                      </div>
                      <div className="flex min-w-0 items-center gap-2">
                        <p className="min-w-0 truncate text-sm font-medium">
                          {name}
                        </p>
                        <Badge
                          variant="secondary"
                          size="sm"
                          className="shrink-0"
                        >
                          {kind === 'direct' ? 'Direct' : 'Indirect'}
                        </Badge>
                      </div>
                    </div>
                    <div className="flex shrink-0 items-center gap-2">
                      <Badge variant="secondary" className="font-mono">
                        {range}
                      </Badge>
                      <ExternalLink className="h-3.5 w-3.5 text-muted-foreground" />
                    </div>
                  </Link>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
