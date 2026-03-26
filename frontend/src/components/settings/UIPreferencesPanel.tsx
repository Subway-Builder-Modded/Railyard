import { ChevronDown, LayoutGrid, Monitor, Palette } from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';

import {
  BROWSE_CONTENT_CN,
  BROWSE_ITEM_CN,
  BROWSE_TRIGGER_CN,
  ControlWrapper,
} from '@/components/settings/ControlWrapper';
import { SettingRow } from '@/components/settings/SettingRow';
import { ThemePicker, type ThemeValue } from '@/components/shared/ThemePicker';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  isSearchViewMode,
  normalizeSearchViewMode,
} from '@/lib/search-view-mode';
import { cn } from '@/lib/utils';
import { useProfileStore } from '@/stores/profile-store';

const VALID_THEMES = new Set<ThemeValue>([
  'dark',
  'light',
  'system',
  'midnight',
  'coffee',
  'forest',
  'crystal',
]);
const THEME_LABELS: Record<ThemeValue, string> = {
  dark: 'Dark',
  light: 'Light',
  system: 'System',
  midnight: 'Midnight',
  coffee: 'Coffee',
  forest: 'Forest',
  crystal: 'Crystal',
};

function normalizeThemeValue(theme: unknown): ThemeValue {
  if (
    theme === 'system' ||
    theme === 'light' ||
    theme === 'dark' ||
    theme === 'midnight' ||
    theme === 'coffee' ||
    theme === 'forest' ||
    theme === 'crystal'
  )
    return theme;
  if (typeof theme === 'string') {
    const lowered = theme.toLowerCase();
    if (lowered.startsWith('light')) return 'light';
    if (lowered.startsWith('dark')) return 'dark';
  }
  return 'dark';
}

export function UIPreferencesPanel() {
  const profile = useProfileStore((s) => s.profile);
  const updateUIPreferences = useProfileStore((s) => s.updateUIPreferences);

  const [showThemePreviews, setShowThemePreviews] = useState(false);

  const handleThemeChange = async (theme: ThemeValue) => {
    if (!profile || !VALID_THEMES.has(theme)) return;
    try {
      await updateUIPreferences({ theme });
      toast.success('Theme updated.');
    } catch {
      toast.error('Failed to update theme.');
    }
  };

  const handleDefaultPerPageChange = async (value: string) => {
    if (!profile) return;
    const parsed = Number.parseInt(value, 10);
    if (![12, 24, 48].includes(parsed)) return;
    try {
      await updateUIPreferences({ defaultPerPage: parsed });
      toast.success('Default cards per page updated.');
    } catch {
      toast.error('Failed to update default cards per page.');
    }
  };

  const handleDefaultBrowseViewModeChange = async (value: string) => {
    if (!profile) {
      console.warn(
        '[settings] Cannot update default browse view mode: profile is not loaded.',
      );
      return;
    }
    if (!isSearchViewMode(value)) {
      console.warn(
        `[settings] Ignoring invalid browse view mode value: ${String(value)}`,
      );
      return;
    }
    try {
      await updateUIPreferences({ searchViewMode: value });
      toast.success('Default browse view mode updated.');
    } catch (error) {
      console.warn(
        '[settings] Failed to persist default browse view mode preference.',
        error,
      );
      toast.error('Failed to update default browse view mode.');
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>UI Preferences</CardTitle>
        <CardDescription>
          Display and layout preferences saved to your profile.
        </CardDescription>
      </CardHeader>
      <CardContent className="px-6 py-0">
        <div className="divide-y divide-border">
          <SettingRow
            icon={<Palette className="h-4 w-4" />}
            label="Theme"
            description="Choose your preferred color scheme"
            action={
              <ControlWrapper>
                <button
                  type="button"
                  onClick={() => setShowThemePreviews((c) => !c)}
                  aria-expanded={showThemePreviews}
                  className={cn(
                    BROWSE_TRIGGER_CN,
                    'flex w-32 items-center justify-between',
                  )}
                >
                  <span>
                    {
                      THEME_LABELS[
                        normalizeThemeValue(profile?.uiPreferences?.theme)
                      ]
                    }
                  </span>
                  <ChevronDown
                    className={cn(
                      'size-3.5 shrink-0 transition-transform duration-200',
                      showThemePreviews && 'rotate-180',
                    )}
                  />
                </button>
              </ControlWrapper>
            }
          >
            {showThemePreviews && (
              <ThemePicker
                value={normalizeThemeValue(profile?.uiPreferences?.theme)}
                onChange={handleThemeChange}
              />
            )}
          </SettingRow>

          <SettingRow
            icon={<LayoutGrid className="h-4 w-4" />}
            label="Default Cards Per Page"
            description="Choose how many items will be shown per page in browse views"
            action={
              <ControlWrapper>
                <Select
                  value={String(profile?.uiPreferences?.defaultPerPage ?? 12)}
                  onValueChange={handleDefaultPerPageChange}
                >
                  <SelectTrigger
                    size="sm"
                    className={cn(BROWSE_TRIGGER_CN, 'w-24')}
                  >
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent className={BROWSE_CONTENT_CN}>
                    <SelectItem value="12" className={BROWSE_ITEM_CN}>
                      12
                    </SelectItem>
                    <SelectItem value="24" className={BROWSE_ITEM_CN}>
                      24
                    </SelectItem>
                    <SelectItem value="48" className={BROWSE_ITEM_CN}>
                      48
                    </SelectItem>
                  </SelectContent>
                </Select>
              </ControlWrapper>
            }
          />

          <SettingRow
            icon={<Monitor className="h-4 w-4" />}
            label="Default Browse View"
            description="Choose the initial layout you will see when opening the browse page"
            action={
              <ControlWrapper>
                <Select
                  value={normalizeSearchViewMode(
                    (
                      profile?.uiPreferences as
                        | { searchViewMode?: unknown }
                        | undefined
                    )?.searchViewMode,
                  )}
                  onValueChange={handleDefaultBrowseViewModeChange}
                >
                  <SelectTrigger
                    size="sm"
                    className={cn(BROWSE_TRIGGER_CN, 'w-28')}
                  >
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent className={BROWSE_CONTENT_CN}>
                    <SelectItem value="full" className={BROWSE_ITEM_CN}>
                      Full
                    </SelectItem>
                    <SelectItem value="compact" className={BROWSE_ITEM_CN}>
                      Compact
                    </SelectItem>
                    <SelectItem value="list" className={BROWSE_ITEM_CN}>
                      List
                    </SelectItem>
                  </SelectContent>
                </Select>
              </ControlWrapper>
            }
          />
        </div>
      </CardContent>
    </Card>
  );
}
