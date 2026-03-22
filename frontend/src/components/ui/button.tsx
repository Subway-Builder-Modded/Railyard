import { cva, type VariantProps } from 'class-variance-authority';
import { Slot } from 'radix-ui';
import * as React from 'react';

import { cn } from '@/lib/utils';

type LegacyVariant =
  | 'default'
  | 'secondary'
  | 'destructive'
  | 'outline'
  | 'ghost'
  | 'link';

type LegacySize =
  | 'default'
  | 'xs'
  | 'sm'
  | 'lg'
  | 'icon'
  | 'icon-xs'
  | 'icon-sm'
  | 'icon-lg';

const buttonStyles = cva(
  [
    '[--btn-border:color-mix(in_oklab,var(--color-foreground)_15%,transparent)]',
    '[--btn-outline:var(--color-ring)]',
    '[--btn-ring:color-mix(in_oklab,var(--color-foreground)_16%,transparent)]',
    '[--btn-bg:transparent] [--btn-fg:var(--color-foreground)] [--btn-overlay:color-mix(in_oklab,var(--color-foreground)_6%,transparent)]',
    'bg-(--btn-bg) text-(--btn-fg) outline-(--btn-outline) ring-(--btn-ring) hover:bg-(--btn-overlay)',
    'relative isolate inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap border border-(--btn-border) font-medium',
    'transition-[background-color,border-color,box-shadow,transform,color] hover:no-underline active:translate-y-px',
    'focus:outline-0 focus-visible:outline focus-visible:outline-offset-2 focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-background',
    'disabled:pointer-events-none disabled:opacity-50',
    '[&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg]:text-current',
  ],
  {
    variants: {
      intent: {
        primary: [
          '[--btn-bg:var(--color-primary)] [--btn-fg:var(--color-primary-foreground)]',
          '[--btn-overlay:color-mix(in_oklab,var(--color-primary-foreground)_10%,var(--color-primary)_90%)]',
          '[--btn-ring:color-mix(in_oklab,var(--color-primary-foreground)_20%,transparent)]',
        ],
        secondary: [
          '[--btn-bg:var(--color-secondary)] [--btn-fg:var(--color-secondary-foreground)]',
          '[--btn-overlay:color-mix(in_oklab,var(--color-secondary-foreground)_10%,var(--color-secondary)_90%)]',
          '[--btn-ring:color-mix(in_oklab,var(--color-muted-foreground)_18%,transparent)]',
        ],
        outline: [
          'border-border [--btn-bg:transparent] [--btn-fg:var(--color-foreground)]',
          '[--btn-overlay:color-mix(in_oklab,var(--color-foreground)_6%,var(--color-secondary)_94%)]',
          '[--btn-ring:color-mix(in_oklab,var(--color-ring)_22%,transparent)]',
        ],
        plain: [
          'border-transparent [--btn-bg:transparent] [--btn-fg:var(--color-foreground)]',
          '[--btn-overlay:color-mix(in_oklab,var(--color-foreground)_6%,transparent)]',
        ],
        danger: [
          '[--btn-bg:var(--uninstall-primary)] [--btn-fg:var(--uninstall-foreground)]',
          '[--btn-overlay:color-mix(in_oklab,var(--uninstall-foreground)_10%,var(--uninstall-primary)_90%)]',
          '[--btn-ring:color-mix(in_oklab,var(--uninstall-primary)_22%,transparent)]',
        ],
        link: [
          'border-transparent [--btn-bg:transparent] [--btn-fg:var(--color-primary)]',
          '[--btn-overlay:transparent] underline underline-offset-4 hover:opacity-90 active:translate-y-0',
        ],
      },
      size: {
        xs: ['min-h-8 gap-x-1.5 px-2.5 py-1.5 text-xs', '[&_svg]:size-3.5'],
        sm: ['min-h-9 gap-x-1.5 px-3 py-2 text-sm', '[&_svg]:size-4'],
        md: ['min-h-10 gap-x-2 px-3.5 py-2.5 text-sm', '[&_svg]:size-4.5'],
        lg: ['min-h-11 gap-x-2 px-4 py-3 text-sm', '[&_svg]:size-5'],
        'sq-xs': ['size-8', '[&_svg]:size-3.5'],
        'sq-sm': ['size-10', '[&_svg]:size-4.5'],
        'sq-md': ['size-11', '[&_svg]:size-5'],
        'sq-lg': ['size-12', '[&_svg]:size-6'],
      },
      isCircle: {
        true: 'rounded-full',
        false: 'rounded-lg',
      },
    },
    defaultVariants: {
      intent: 'primary',
      size: 'md',
      isCircle: false,
    },
  },
);

type ButtonStyleProps = VariantProps<typeof buttonStyles>;

export interface ButtonProps
  extends React.ComponentProps<'button'>, Omit<ButtonStyleProps, 'size'> {
  asChild?: boolean;
  intent?: ButtonStyleProps['intent'];
  variant?: LegacyVariant;
  size?: ButtonStyleProps['size'] | LegacySize;
}

const variantToIntent: Record<
  LegacyVariant,
  NonNullable<ButtonStyleProps['intent']>
> = {
  default: 'primary',
  secondary: 'secondary',
  destructive: 'danger',
  outline: 'outline',
  ghost: 'plain',
  link: 'link',
};

const sizeAlias: Record<LegacySize, NonNullable<ButtonStyleProps['size']>> = {
  default: 'md',
  xs: 'xs',
  sm: 'sm',
  lg: 'lg',
  icon: 'sq-sm',
  'icon-xs': 'sq-xs',
  'icon-sm': 'sq-sm',
  'icon-lg': 'sq-lg',
};

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    { className, variant, intent, size, isCircle, asChild = false, ...props },
    ref,
  ) => {
    const Comp = asChild ? Slot.Root : 'button';
    const resolvedIntent =
      intent ?? (variant ? variantToIntent[variant] : undefined);
    const resolvedSize = size
      ? (sizeAlias[size as LegacySize] ?? (size as ButtonStyleProps['size']))
      : undefined;

    return (
      <Comp
        ref={ref}
        data-slot="button"
        data-variant={variant}
        className={cn(
          buttonStyles({
            intent: resolvedIntent,
            size: resolvedSize,
            isCircle,
          }),
          className,
        )}
        {...props}
      />
    );
  },
);
Button.displayName = 'Button';

export { Button, buttonStyles };
