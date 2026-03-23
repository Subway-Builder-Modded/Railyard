import { ToggleLeft, ToggleRight } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

const TOGGLE_ENABLED: Record<'install' | 'update', string> = {
  install:
    'border-[color-mix(in_oklab,var(--install-primary)_40%,transparent)] text-[var(--install-primary)] hover:bg-[color-mix(in_oklab,var(--install-primary)_8%,transparent)] hover:text-[var(--install-primary)]',
  update:
    'border-[color-mix(in_oklab,var(--update-primary)_40%,transparent)] text-[var(--update-primary)] hover:bg-[color-mix(in_oklab,var(--update-primary)_8%,transparent)] hover:text-[var(--update-primary)]',
};

interface SettingToggleButtonProps {
  enabled: boolean;
  onToggle: () => void;
  accent?: 'install' | 'update';
  enabledLabel?: string;
  disabledLabel?: string;
}

export function SettingToggleButton({
  enabled,
  onToggle,
  accent = 'install',
  enabledLabel = 'Enabled',
  disabledLabel = 'Disabled',
}: SettingToggleButtonProps) {
  return (
    <Button
      variant="outline"
      size="sm"
      onClick={onToggle}
      className={cn('min-w-[6rem] gap-1.5 transition-all', enabled && TOGGLE_ENABLED[accent])}
    >
      {enabled ? (
        <ToggleRight className="size-3.5 shrink-0" />
      ) : (
        <ToggleLeft className="size-3.5 shrink-0" />
      )}
      {enabled ? enabledLabel : disabledLabel}
    </Button>
  );
}
