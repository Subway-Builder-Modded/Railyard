import type { ComponentType } from 'react';

import { cn } from '@/lib/utils';

interface SectionHeaderProps {
  title: string;
  icon?: ComponentType<{ className?: string }>;
  badge?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
}

export function SectionHeader({
  title,
  icon: Icon,
  badge,
  action,
  className,
}: SectionHeaderProps) {
  return (
    <div
      className={cn(
        'mb-[clamp(0.65rem,1.2vw,0.9rem)] flex items-center justify-between',
        className,
      )}
    >
      <div className="flex items-center gap-2.5">
        {Icon ? (
          <Icon
            className="size-[1.05rem] shrink-0 text-foreground"
            aria-hidden
          />
        ) : null}
        <h2 className="text-[clamp(0.95rem,1.5vw,1.1rem)] font-bold tracking-tight text-foreground">
          {title}
        </h2>
        {badge}
      </div>
      {action}
    </div>
  );
}
