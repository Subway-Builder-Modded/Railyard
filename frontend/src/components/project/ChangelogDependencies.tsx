import { ExternalLink, Gamepad2, Loader2, Package } from 'lucide-react';
import { useEffect, useState } from 'react';
import { Link } from 'wouter';

import { EmptyState } from '@/components/shared/EmptyState';
import { Badge } from '@/components/ui/badge';
import type { AssetType } from '@/lib/asset-types';
import { assetTypeToListingPath } from '@/lib/asset-types';
import { useRegistryStore } from '@/stores/registry-store';

import { ComputeDependencyList } from '../../../wailsjs/go/downloader/Downloader';
import type { types } from '../../../wailsjs/go/models';

const GAME_DEP_KEY = 'subway-builder';

interface ChangelogDependenciesProps {
  type: AssetType;
  itemId: string;
  versionInfo: types.VersionInfo;
}

interface FlatDep {
  id: string;
  range: string;
  kind: 'direct' | 'indirect';
}

export function ChangelogDependencies({
  type,
  itemId,
  versionInfo,
}: ChangelogDependenciesProps) {
  const mods = useRegistryStore((s) => s.mods);
  const [resolvedDeps, setResolvedDeps] = useState<Record<
    string,
    types.DependencyListEntry
  > | null>(null);
  const [resolving, setResolving] = useState(false);

  const rawDeps = versionInfo.dependencies ?? {};
  const hasModDeps = Object.keys(rawDeps).some((k) => k !== GAME_DEP_KEY);

  useEffect(() => {
    if (type !== 'mod' || !hasModDeps) {
      setResolvedDeps(null);
      return;
    }
    let cancelled = false;
    setResolving(true);
    ComputeDependencyList(itemId, versionInfo)
      .then((result) => {
        if (cancelled) return;
        const list = { ...(result.installList ?? {}) };
        delete list[itemId];
        setResolvedDeps(list);
        setResolving(false);
      })
      .catch(() => {
        if (!cancelled) {
          setResolvedDeps(null);
          setResolving(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [type, itemId, versionInfo.version]);

  const gameDep = rawDeps[GAME_DEP_KEY];
  const directIds = new Set(
    Object.keys(rawDeps).filter((k) => k !== GAME_DEP_KEY),
  );

  const flatDeps: FlatDep[] = [];
  if (resolvedDeps !== null) {
    for (const id of directIds) {
      flatDeps.push({ id, range: rawDeps[id], kind: 'direct' });
    }
    for (const [id, entry] of Object.entries(resolvedDeps)) {
      if (!directIds.has(id)) {
        flatDeps.push({
          id,
          range: entry.installCandidate.version,
          kind: 'indirect',
        });
      }
    }
  } else {
    for (const id of directIds) {
      flatDeps.push({ id, range: rawDeps[id], kind: 'direct' });
    }
  }

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
              <Gamepad2 className="h-4 w-4 text-primary" />
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
                      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-muted">
                        <Package className="h-3.5 w-3.5 text-muted-foreground" />
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
