import { cn } from '@/lib/utils';

export interface SettingRowProps {
  icon: React.ReactNode;
  iconClassName?: string;
  label: string;
  badge?: React.ReactNode;
  description?: React.ReactNode;
  action: React.ReactNode;
  children?: React.ReactNode;
}

export function SettingRow({
  icon,
  iconClassName,
  label,
  badge,
  description,
  action,
  children,
}: SettingRowProps) {
  return (
    <div className="py-4">
      <div className="flex items-center gap-3.5">
        <div
          className={cn(
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground',
            iconClassName,
          )}
        >
          {icon}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <p className="text-sm font-medium leading-snug">{label}</p>
            {badge}
          </div>
          {description && (
            <p className="mt-0.5 text-xs leading-snug text-muted-foreground">{description}</p>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-2">{action}</div>
      </div>
      {children && <div className="mt-3 ml-11">{children}</div>}
    </div>
  );
}
