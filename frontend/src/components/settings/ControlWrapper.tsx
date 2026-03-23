import { cn } from '@/lib/utils';

export const BROWSE_TRIGGER_CN = cn(
  'border-0 bg-transparent shadow-none',
  'dark:bg-transparent dark:hover:bg-transparent',
  'h-8 gap-2 px-3',
  'text-xs font-semibold text-muted-foreground',
  'hover:bg-accent/45 hover:text-primary dark:hover:bg-accent/45',
  'data-[state=open]:bg-accent/45 data-[state=open]:text-primary',
  '[&_svg]:!text-current [&_svg:last-child]:!opacity-100',
);

export const BROWSE_CONTENT_CN =
  'rounded-xl border border-border/70 bg-background/95 p-1 shadow-lg backdrop-blur-md';

export const BROWSE_ITEM_CN = cn(
  'rounded-lg text-sm',
  'data-[highlighted]:bg-accent/45 data-[highlighted]:text-primary',
  'data-[state=checked]:bg-accent/35 data-[state=checked]:text-primary',
);

export function ControlWrapper({ children }: { children: React.ReactNode }) {
  return (
    <div className="overflow-hidden rounded-xl border border-border/70 bg-background/90 shadow-sm backdrop-blur-md">
      {children}
    </div>
  );
}
