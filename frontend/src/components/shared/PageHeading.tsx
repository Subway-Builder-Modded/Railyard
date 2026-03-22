import type { LucideIcon } from 'lucide-react';

import { cn } from '@/lib/utils';

export type PageHeadingSize = 'default' | 'compact' | 'sidebar';

export interface PageHeadingProps {
  icon: LucideIcon;
  title: string;
  description?: string;
  size?: PageHeadingSize;
  className?: string;
}

export function PageHeading({
  icon: Icon,
  title,
  description,
  size = 'default',
  className,
}: PageHeadingProps) {
  const isCompact = size === 'compact' || size === 'sidebar';
  const isSidebar = size === 'sidebar';

  return (
    <header className={cn('relative mb-10 text-center', className)}>
      <div
        className={cn(
          'pointer-events-none absolute inset-x-0 mx-auto rounded-full blur-3xl',
          isSidebar
            ? '-top-7 h-24 max-w-sm'
            : isCompact
              ? '-top-7 h-24 max-w-sm'
              : '-top-10 h-36 w-full',
          'bg-gradient-to-r from-transparent via-primary/20 to-transparent',
        )}
      />

      <h1
        className={cn(
          'relative mt-1 inline-flex items-center justify-center font-black tracking-tight',
          isSidebar
            ? 'gap-2.5 text-[2.35rem] leading-tight'
            : isCompact
              ? 'gap-2 text-2xl'
              : 'gap-3 text-4xl sm:text-5xl',
        )}
      >
        <Icon
          className={cn(
            'text-foreground',
            isSidebar
              ? 'size-[1em]'
              : isCompact
                ? 'size-[0.85em]'
                : 'size-[0.95em]',
          )}
          aria-hidden="true"
        />
        <span>{title}</span>
      </h1>

      {description ? (
        <p
          className={cn(
            'text-muted-foreground',
            isCompact
              ? 'mt-2 max-w-xs text-xs leading-4'
              : 'mt-3 w-full text-base sm:text-lg',
          )}
        >
          {description}
        </p>
      ) : null}
    </header>
  );
}
