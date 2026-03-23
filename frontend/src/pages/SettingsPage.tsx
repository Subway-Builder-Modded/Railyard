import { Settings } from 'lucide-react';
import { useState } from 'react';

import { DangerZonePanel } from '@/components/settings/DangerZonePanel';
import { GeneralSettingsPanel } from '@/components/settings/GeneralSettingsPanel';
import { SettingsNav, type SettingsTab } from '@/components/settings/SettingsNav';
import { SystemPreferencesPanel } from '@/components/settings/SystemPreferencesPanel';
import { UIPreferencesPanel } from '@/components/settings/UIPreferencesPanel';
import { PageHeading } from '@/components/shared/PageHeading';

export function SettingsPage() {
  const [activeTab, setActiveTab] = useState<SettingsTab>('general');

  return (
    <div className="space-y-6">
      <PageHeading
        icon={Settings}
        title="Settings"
        description="Configure Railyard and customize your experience."
      />

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
