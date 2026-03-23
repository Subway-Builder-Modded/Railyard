import { toast } from 'sonner';

import type { InstalledTaggedItem } from '@/hooks/use-filtered-installed-items';
import { joinOsPath } from '@/lib/utils';

import { OpenInFileExplorer } from '../../wailsjs/go/main/App';
import type { types } from '../../wailsjs/go/models';

export function resolveInstallFolderPath(
  entry: InstalledTaggedItem,
  metroMakerDataPath: string | undefined,
): string | null {
  if (!metroMakerDataPath) return null;
  if (entry.type === 'mod') return joinOsPath(metroMakerDataPath, 'mods', entry.item.id);
  const cityCode = ((entry.item as types.MapManifest).city_code ?? '').trim();
  return cityCode
    ? joinOsPath(metroMakerDataPath, 'cities', 'data', cityCode)
    : joinOsPath(metroMakerDataPath, 'cities', 'data');
}

export function openInstallFolder(
  entry: InstalledTaggedItem,
  metroMakerDataPath: string | undefined,
): void {
  const path = resolveInstallFolderPath(entry, metroMakerDataPath);
  if (!path) return;
  void (async () => {
    try {
      const result = await OpenInFileExplorer(path);
      if (result?.status === 'error') {
        toast.error(result?.message || 'Failed to open install folder');
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  })();
}
