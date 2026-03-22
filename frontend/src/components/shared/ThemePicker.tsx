import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

export type ThemeValue = 'dark' | 'light' | 'system';

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
];

interface ThemePreviewProps {
  colors: ThemePreviewColors;
}

function ThemePreview({ colors }: ThemePreviewProps) {
  return (
    <div
      className="relative w-full rounded-sm overflow-hidden"
      style={{
        aspectRatio: '16 / 10',
        background: colors.bg,
        border: `1px solid ${colors.border}`,
      }}
    >
      {/* Top bar */}
      <div
        className="absolute top-0 left-0 right-0 flex items-center gap-1 px-2"
        style={{
          height: '15%',
          background: colors.bar,
          borderBottom: `1px solid ${colors.border}`,
        }}
      >
        <div
          className="rounded-full"
          style={{ width: 5, height: 5, background: '#f47067' }}
        />
        <div
          className="rounded-full"
          style={{ width: 5, height: 5, background: '#f1a33c' }}
        />
        <div
          className="rounded-full"
          style={{ width: 5, height: 5, background: '#58be40' }}
        />
        <div
          className="ml-2 rounded-sm flex-1"
          style={{ height: 6, background: colors.muted, maxWidth: '40%' }}
        />
        <div className="ml-auto flex items-center gap-1">
          <div
            className="rounded-[2px]"
            style={{ width: 6, height: 6, background: '#2da44e' }}
          />
          <div
            className="rounded-[2px]"
            style={{ width: 6, height: 6, background: '#f85149' }}
          />
        </div>
      </div>

      {/* Body */}
      <div
        className="absolute bottom-0 left-0 right-0 flex"
        style={{ top: '15%' }}
      >
        {/* Sidebar strip */}
        <div
          className="flex flex-col gap-1 p-1.5"
          style={{
            width: '30%',
            background: colors.sidebar,
            borderRight: `1px solid ${colors.border}`,
          }}
        >
          <div
            className="rounded-sm"
            style={{ height: 5, background: colors.primary, width: '70%' }}
          />
          <div
            className="rounded-sm"
            style={{ height: 5, background: colors.muted, width: '90%' }}
          />
          <div
            className="rounded-sm"
            style={{ height: 5, background: colors.muted, width: '60%' }}
          />
          <div
            className="rounded-sm"
            style={{ height: 5, background: colors.muted, width: '80%' }}
          />
        </div>

        {/* Main content */}
        <div className="flex-1 p-2 flex flex-col gap-1.5">
          <div
            className="rounded-sm"
            style={{ height: 4, width: '42%', background: '#2da44e' }}
          />
          <div
            className="rounded-sm"
            style={{ height: 7, background: colors.muted, width: '65%' }}
          />
          <div
            className="rounded-sm"
            style={{
              height: 20,
              background: colors.card,
              border: `1px solid ${colors.border}`,
            }}
          />
          <div className="flex gap-1.5">
            <div
              className="rounded-sm flex-1"
              style={{
                height: 12,
                background: colors.card,
                border: `1px solid ${colors.border}`,
              }}
            />
            <div
              className="rounded-sm flex-1"
              style={{
                height: 12,
                background: colors.card,
                border: `1px solid ${colors.border}`,
              }}
            />
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
    <div className="grid grid-cols-2 gap-3">
      {THEME_OPTIONS.map((option) => {
        const isSelected = value === option.value;
        return (
          <Button
            key={option.value}
            type="button"
            disabled={disabled}
            onClick={() => onChange(option.value)}
            intent="plain"
            size="md"
            className={cn(
              'group h-auto flex-col items-stretch gap-2 border p-2 text-left transition-all active:translate-y-0',
              'disabled:cursor-not-allowed disabled:opacity-50',
              isSelected
                ? 'border-foreground/25 ring-2 ring-foreground/10 bg-muted/15'
                : 'border-border hover:border-foreground/20 hover:bg-muted/20',
            )}
            aria-pressed={isSelected}
          >
            <ThemePreview colors={option.colors} />
            <div className="flex items-center gap-2 px-0.5">
              <div
                className={cn(
                  'flex h-3.5 w-3.5 shrink-0 items-center justify-center rounded-full border-2 transition-colors',
                  isSelected
                    ? 'border-foreground/60'
                    : 'border-muted-foreground/40 group-hover:border-foreground/45',
                )}
              >
                {isSelected && (
                  <div className="h-1.5 w-1.5 rounded-full bg-foreground/80" />
                )}
              </div>
              <span className="text-xs font-medium leading-none">
                {option.label}
              </span>
            </div>
          </Button>
        );
      })}
    </div>
  );
}
