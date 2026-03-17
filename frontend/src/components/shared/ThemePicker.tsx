import { cn } from '@/lib/utils';

export type ThemeValue =
  | 'dark'
  | 'dark_low'
  | 'dark_high'
  | 'light'
  | 'light_low'
  | 'light_high'
  | 'system';

interface ThemePreviewColors {
  bg: string;
  sidebar: string;
  card: string;
  bar: string;
  primary: string;
  muted: string;
  border: string;
}

interface ThemeOption {
  value: ThemeValue;
  label: string;
  colors: ThemePreviewColors;
}

const THEME_OPTIONS: ThemeOption[] = [
  {
    value: 'light',
    label: 'Light',
    colors: {
      bg: '#fafafa',
      sidebar: '#f7f7f7',
      card: '#ffffff',
      bar: '#f3f3f3',
      primary: '#3f3f3f',
      muted: '#ececec',
      border: '#dfdfdf',
    },
  },
  {
    value: 'light_low',
    label: 'Light (Soft)',
    colors: {
      bg: '#d4cec4',
      sidebar: '#cbc5bb',
      card: '#ddd6cd',
      bar: '#bfb8ad',
      primary: '#6f665b',
      muted: '#b4ad9f',
      border: '#9d9587',
    },
  },
  {
    value: 'light_high',
    label: 'Light (Contrast)',
    colors: {
      bg: '#ffffff',
      sidebar: '#fbfbfb',
      card: '#ffffff',
      bar: '#efefef',
      primary: '#0f0f0f',
      muted: '#d9d9d9',
      border: '#737373',
    },
  },
  {
    value: 'dark',
    label: 'Dark',
    colors: {
      bg: '#1c1c1c',
      sidebar: '#232323',
      card: '#282828',
      bar: '#1c1c1c',
      primary: '#ebebeb',
      muted: '#383838',
      border: '#2e2e2e',
    },
  },
  {
    value: 'dark_low',
    label: 'Dark (Soft)',
    colors: {
      bg: '#3a3a3a',
      sidebar: '#434343',
      card: '#4b4b4b',
      bar: '#3f3f3f',
      primary: '#e4e4e4',
      muted: '#585858',
      border: '#666666',
    },
  },
  {
    value: 'dark_high',
    label: 'Dark (Contrast)',
    colors: {
      bg: '#050505',
      sidebar: '#0c0c0c',
      card: '#121212',
      bar: '#080808',
      primary: '#ffffff',
      muted: '#232323',
      border: '#6a6a6a',
    },
  },
];

interface ThemePreviewProps {
  colors: ThemePreviewColors;
}

function ThemePreview({ colors }: ThemePreviewProps) {
  return (
    <div
      className="relative w-full rounded-sm overflow-hidden"
      style={{ aspectRatio: '16 / 10', background: colors.bg, border: `1px solid ${colors.border}` }}
    >
      {/* Top bar */}
      <div
        className="absolute top-0 left-0 right-0 flex items-center gap-1 px-2"
        style={{ height: '15%', background: colors.bar, borderBottom: `1px solid ${colors.border}` }}
      >
        <div className="rounded-full" style={{ width: 5, height: 5, background: '#f47067' }} />
        <div className="rounded-full" style={{ width: 5, height: 5, background: '#f1a33c' }} />
        <div className="rounded-full" style={{ width: 5, height: 5, background: '#58be40' }} />
        <div
          className="ml-2 rounded-sm flex-1"
          style={{ height: 6, background: colors.muted, maxWidth: '40%' }}
        />
        <div className="ml-auto flex items-center gap-1">
          <div className="rounded-[2px]" style={{ width: 6, height: 6, background: '#2da44e' }} />
          <div className="rounded-[2px]" style={{ width: 6, height: 6, background: '#f85149' }} />
        </div>
      </div>

      {/* Body */}
      <div className="absolute bottom-0 left-0 right-0 flex" style={{ top: '15%' }}>
        {/* Sidebar strip */}
        <div
          className="flex flex-col gap-1 p-1.5"
          style={{ width: '30%', background: colors.sidebar, borderRight: `1px solid ${colors.border}` }}
        >
          <div className="rounded-sm" style={{ height: 5, background: colors.primary, width: '70%' }} />
          <div className="rounded-sm" style={{ height: 5, background: colors.muted, width: '90%' }} />
          <div className="rounded-sm" style={{ height: 5, background: colors.muted, width: '60%' }} />
          <div className="rounded-sm" style={{ height: 5, background: colors.muted, width: '80%' }} />
        </div>

        {/* Main content */}
        <div className="flex-1 p-2 flex flex-col gap-1.5">
          <div
            className="rounded-sm"
            style={{ height: 4, width: '42%', background: '#2da44e' }}
          />
          <div className="rounded-sm" style={{ height: 7, background: colors.muted, width: '65%' }} />
          <div
            className="rounded-sm"
            style={{ height: 20, background: colors.card, border: `1px solid ${colors.border}` }}
          />
          <div className="flex gap-1.5">
            <div className="rounded-sm flex-1" style={{ height: 12, background: colors.card, border: `1px solid ${colors.border}` }} />
            <div className="rounded-sm flex-1" style={{ height: 12, background: colors.card, border: `1px solid ${colors.border}` }} />
          </div>
        </div>
      </div>
    </div>
  );
}

interface ThemePickerProps {
  value: ThemeValue;
  onChange: (theme: ThemeValue) => void;
  disabled?: boolean;
}

export function ThemePicker({ value, onChange, disabled }: ThemePickerProps) {
  return (
    <div className="grid grid-cols-3 gap-3">
      {THEME_OPTIONS.map((option) => {
        const isSelected = value === option.value;
        return (
          <button
            key={option.value}
            type="button"
            disabled={disabled}
            onClick={() => onChange(option.value)}
            className={cn(
              'group flex flex-col gap-2 rounded-lg border p-2 text-left transition-all',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
              'disabled:cursor-not-allowed disabled:opacity-50',
              isSelected
                ? 'border-primary ring-2 ring-primary/40 bg-primary/5'
                : 'border-border hover:border-primary/40 hover:bg-accent/50',
            )}
            aria-pressed={isSelected}
          >
            <ThemePreview colors={option.colors} />
            <div className="flex items-center gap-2 px-0.5">
              <div
                className={cn(
                  'flex h-3.5 w-3.5 shrink-0 items-center justify-center rounded-full border-2 transition-colors',
                  isSelected
                    ? 'border-primary'
                    : 'border-muted-foreground/40 group-hover:border-primary/60',
                )}
              >
                {isSelected && (
                  <div className="h-1.5 w-1.5 rounded-full bg-primary" />
                )}
              </div>
              <span className="text-xs font-medium leading-none">{option.label}</span>
            </div>
          </button>
        );
      })}
    </div>
  );
}
