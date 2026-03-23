import { AlertTriangle, Cpu, Palette, Settings } from 'lucide-react';

import { cn } from '@/lib/utils';

export type SettingsTab = 'general' | 'ui' | 'system' | 'danger';

const NAV_ITEMS: Array<{
  id: SettingsTab;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  description: string;
  danger?: boolean;
}> = [
  { id: 'general', label: 'General', icon: Settings, description: 'General application settings' },
  { id: 'ui', label: 'UI Preferences', icon: Palette, description: 'Theme & display' },
  { id: 'system', label: 'System', icon: Cpu, description: 'Performance & tools' },
  { id: 'danger', label: 'Danger Zone', icon: AlertTriangle, description: 'Reset & clear', danger: true },
];

interface SettingsNavProps {
  activeTab: SettingsTab;
  onTabChange: (tab: SettingsTab) => void;
}

export function SettingsNav({ activeTab, onTabChange }: SettingsNavProps) {
  return (
    <nav className="w-48 shrink-0 flex flex-col gap-0.5">
      {NAV_ITEMS.map((item) => {
        const Icon = item.icon;
        const isActive = activeTab === item.id;
        return (
          <button
            key={item.id}
            onClick={() => onTabChange(item.id)}
            className={cn(
              'group flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left text-sm font-medium transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1',
              isActive
                ? item.danger
                  ? 'bg-destructive/10 text-destructive shadow-sm'
                  : 'bg-muted text-foreground shadow-sm'
                : item.danger
                  ? 'text-muted-foreground hover:bg-destructive/8 hover:text-destructive'
                  : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
            )}
          >
            <span
              className={cn(
                'flex h-7 w-7 shrink-0 items-center justify-center rounded-md transition-all duration-150',
                isActive
                  ? item.danger
                    ? 'bg-destructive/15 text-destructive'
                    : 'bg-background text-foreground shadow-sm'
                  : item.danger
                    ? 'bg-destructive/8 text-destructive/60 group-hover:bg-destructive/12 group-hover:text-destructive'
                    : 'bg-background/60 text-muted-foreground group-hover:bg-background group-hover:text-foreground',
              )}
            >
              <Icon className="h-3.5 w-3.5" />
            </span>
            <div className="min-w-0 flex-1">
              <span className="block truncate text-[13px] leading-snug">{item.label}</span>
              <span
                className={cn(
                  'mt-1 block text-[11px] leading-tight',
                  isActive
                    ? item.danger
                      ? 'text-destructive/60'
                      : 'text-muted-foreground'
                    : 'text-muted-foreground/60 group-hover:text-muted-foreground',
                )}
              >
                {item.description}
              </span>
            </div>
          </button>
        );
      })}
    </nav>
  );
}
