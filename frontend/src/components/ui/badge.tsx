import { cva, type VariantProps } from 'class-variance-authority';
import { Slot } from 'radix-ui';
import * as React from 'react';

import { cn } from '@/lib/utils';

const badgeVariants = cva(
  [
    'inline-flex w-fit shrink-0 items-center justify-center gap-1 overflow-hidden whitespace-nowrap rounded-full border px-2 py-0.5 text-xs font-medium',
    'border-border/70 bg-muted/35 text-foreground',
    'transition-[background-color,border-color,color,box-shadow]',
    'focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50',
    '[&>svg]:pointer-events-none [&>svg]:size-3',
  ],
  {
    variants: {
      variant: {
        default: '[a&]:hover:bg-muted/55',
        secondary:
          'bg-background/50 text-muted-foreground [a&]:hover:bg-muted/40',
        success:
          'border-[color-mix(in_oklab,var(--install-primary)_35%,transparent)] bg-[color-mix(in_oklab,var(--install-primary)_14%,transparent)] text-[var(--install-primary)] [a&]:hover:bg-[color-mix(in_oklab,var(--install-primary)_22%,transparent)]',
        destructive:
          'border-[color-mix(in_oklab,var(--uninstall-primary)_35%,transparent)] bg-[color-mix(in_oklab,var(--uninstall-primary)_14%,transparent)] text-[var(--uninstall-primary)] [a&]:hover:bg-[color-mix(in_oklab,var(--uninstall-primary)_22%,transparent)]',
        outline: 'bg-transparent text-muted-foreground [a&]:hover:bg-muted/35',
        ghost:
          'border-transparent bg-transparent text-muted-foreground [a&]:hover:bg-muted/35',
        link: 'border-transparent bg-transparent px-0 text-primary underline-offset-4 [a&]:hover:underline',
      },
      size: {
        sm: 'h-5 px-1.5 text-[11px]',
        md: 'h-6 px-2 text-xs',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'md',
    },
  },
);

function Badge({
  className,
  variant = 'default',
  size = 'md',
  asChild = false,
  ...props
}: React.ComponentProps<'span'> &
  VariantProps<typeof badgeVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot.Root : 'span';

  return (
    <Comp
      data-slot="badge"
      data-variant={variant}
      data-size={size}
      className={cn(badgeVariants({ variant, size }), className)}
      {...props}
    />
  );
}

export { Badge, badgeVariants };
