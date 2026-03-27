import { CircleUser, Settings } from 'lucide-react';
import { useEffect } from 'react';

import { DangerZonePanel } from '@/components/settings/DangerZonePanel';
import { GeneralSettingsPanel } from '@/components/settings/GeneralSettingsPanel';
import {
  SettingsNav,
  type SettingsTab,
} from '@/components/settings/SettingsNav';
import { SystemPreferencesPanel } from '@/components/settings/SystemPreferencesPanel';
import { UIPreferencesPanel } from '@/components/settings/UIPreferencesPanel';
import { PageHeading } from '@/components/shared/PageHeading';
import { Badge } from '@/components/ui/badge';
import { useProfileStore } from '@/stores/profile-store';
import { useUIStore } from '@/stores/ui-store';

export function SettingsPage() {
  const activeTab = useUIStore((s) => s.settingsTab) as SettingsTab;
  const setActiveTab = useUIStore((s) => s.setSettingsTab);
  const profile = useProfileStore((s) => s.profile);
  const refreshActiveProfile = useProfileStore((s) => s.refreshActiveProfile);

  useEffect(() => {
    void refreshActiveProfile().catch(() => {});
  }, [refreshActiveProfile]);

  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <div className="flex justify-center">
          <Badge
            className="h-7 gap-1.5 border-[color-mix(in_srgb,var(--profiles-primary)_40%,transparent)] bg-[color-mix(in_srgb,var(--profiles-primary)_14%,transparent)] px-3 text-[var(--profiles-primary)]"
            aria-label={`Current profile: ${profile?.name ?? 'Loading profile'}`}
          >
            <CircleUser className="size-3.5" />
            <span className="max-w-[16rem] truncate font-semibold">
              {profile?.name ?? 'Loading profile'}
            </span>
          </Badge>
        </div>
        <PageHeading
          icon={Settings}
          title="Settings"
          description="Configure Railyard and customize your experience."
        />
      </div>

      <div className="relative z-[1] flex items-start gap-5">
        <SettingsNav activeTab={activeTab} onTabChange={setActiveTab} />

        <div className="min-w-0 flex-1">
          {activeTab === 'general' && <GeneralSettingsPanel />}
          {activeTab === 'ui' && <UIPreferencesPanel />}
          {activeTab === 'system' && <SystemPreferencesPanel />}
          {activeTab === 'danger' && <DangerZonePanel />}
        </div>
      </div>
    </div>
  );
}
