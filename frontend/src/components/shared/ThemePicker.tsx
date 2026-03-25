import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

export type ThemeValue =
  | 'dark'
  | 'light'
  | 'system'
  | 'midnight'
  | 'coffee'
  | 'forest'
  | 'crystal';

type PreviewTheme = Exclude<ThemeValue, 'system'>;

interface ThemeOption {
  value: PreviewTheme;
  label: string;
}

const THEME_OPTIONS: ThemeOption[] = [
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
  { value: 'coffee', label: 'Coffee' },
  { value: 'midnight', label: 'Midnight' },
  { value: 'crystal', label: 'Crystal' },
  { value: 'forest', label: 'Forest' },
];

interface ThemeVars {
  background: string;
  card: string;
  border: string;
  muted: string;
  mutedFg: string;
  accent: string;
  primary: string;
  install: string;
  uninstall: string;
  update: string;
  files: string;
}

const THEME_VARS: Record<PreviewTheme, ThemeVars> = {
  light: {
    background: 'oklch(0.98 0 0)',
    card: 'oklch(1 0 0)',
    border: 'oklch(0.88 0 0)',
    muted: 'oklch(0.95 0 0)',
    mutedFg: 'oklch(0.5 0 0)',
    accent: '#dbeee8',
    primary: '#0f8f68',
    install: '#0f8f68',
    uninstall: '#b03a3a',
    update: '#3f6fd1',
    files: '#d4af37',
  },
  dark: {
    background: 'oklch(0.11 0 0)',
    card: 'oklch(0.16 0 0)',
    border: 'oklch(1 0 0 / 8%)',
    muted: 'oklch(0.22 0 0)',
    mutedFg: 'oklch(0.62 0 0)',
    accent: '#0b3327',
    primary: '#32e8b0',
    install: '#32e8b0',
    uninstall: '#b83a5d',
    update: '#3a6bbf',
    files: '#b58721',
  },
  crystal: {
    background: 'oklch(0.96 0.028 192)',
    card: 'oklch(0.98 0.022 192)',
    border: 'oklch(0.77 0.036 192)',
    muted: 'oklch(0.89 0.032 192)',
    mutedFg: 'oklch(0.48 0.042 196)',
    accent: '#b1ddde',
    primary: 'oklch(0.56 0.082 208)',
    install: '#0a9070',
    uninstall: '#b82020',
    update: '#1040c8',
    files: '#b07808',
  },
  coffee: {
    background: 'oklch(0.2 0.026 56)',
    card: 'oklch(0.24 0.026 56)',
    border: 'oklch(0.48 0.024 58 / 42%)',
    muted: 'oklch(0.31 0.028 56)',
    mutedFg: 'oklch(0.68 0.02 74)',
    accent: '#5b3e28',
    primary: 'oklch(0.76 0.066 74)',
    install: '#58c89a',
    uninstall: '#ce6666',
    update: '#7da2d9',
    files: '#d4a95f',
  },
  midnight: {
    background: 'oklch(0.1 0.038 258)',
    card: 'oklch(0.15 0.035 258)',
    border: 'oklch(0.4 0.03 256 / 34%)',
    muted: 'oklch(0.21 0.036 258)',
    mutedFg: 'oklch(0.62 0.02 242)',
    accent: '#1f314f',
    primary: 'oklch(0.74 0.092 250)',
    install: '#28d8a8',
    uninstall: '#c83860',
    update: '#4888e0',
    files: '#b09020',
  },
  forest: {
    background: 'oklch(0.11 0.028 152)',
    card: 'oklch(0.16 0.028 152)',
    border: 'oklch(1 0 0 / 9%)',
    muted: 'oklch(0.2 0.028 152)',
    mutedFg: 'oklch(0.58 0.018 148)',
    accent: '#213e28',
    primary: 'oklch(0.77 0.103 152)',
    install: '#32e8b0',
    uninstall: '#f07060',
    update: '#60a8f8',
    files: '#e8c040',
  },
};

interface ThemePreviewProps {
  theme: PreviewTheme;
}

function ThemePreview({ theme }: ThemePreviewProps) {
  const v = THEME_VARS[theme];

  return (
    <div
      className="relative w-full overflow-hidden rounded-sm"
      style={{
        aspectRatio: '16 / 10',
        background: v.background,
        border: `1px solid ${v.border}`,
      }}
    >
      <div className="absolute inset-0 flex flex-col gap-[3.5%] p-[5%]">
        {/* Navbar */}
        <div
          className="flex shrink-0 items-center justify-between rounded-[3px] px-1.5"
          style={{
            height: '19%',
            background: v.card,
            border: `1px solid ${v.border}`,
          }}
        >
          <div className="flex items-center gap-1">
            <div
              className="rounded-[2px]"
              style={{ width: 9, height: 6, background: v.primary }}
            />
            <div
              className="rounded-[2px]"
              style={{ width: 15, height: 3.5, background: v.muted }}
            />
            <div
              className="rounded-[2px]"
              style={{
                width: 12,
                height: 3.5,
                background: v.muted,
                opacity: 0.5,
              }}
            />
            <div
              className="rounded-[2px]"
              style={{
                width: 10,
                height: 3.5,
                background: v.muted,
                opacity: 0.5,
              }}
            />
          </div>
          <div className="flex items-center gap-1">
            <div
              className="rounded-[2px]"
              style={{ width: 13, height: 5, background: v.install }}
            />
            <div
              className="rounded-[2px]"
              style={{
                width: 10,
                height: 5,
                background: v.muted,
                opacity: 0.4,
              }}
            />
          </div>
        </div>

        {/* Content — two-column matching the home page layout */}
        <div className="flex flex-1 gap-[3.5%]">
          {/* Left card */}
          <div
            className="flex flex-[3] flex-col gap-[5%] rounded-[3px] p-[5%]"
            style={{ background: v.card, border: `1px solid ${v.border}` }}
          >
            <div
              className="rounded-[1px]"
              style={{ width: '45%', height: 3, background: v.primary }}
            />
            {(
              [
                [85, 0.35],
                [70, 0.28],
                [55, 0.22],
              ] as [number, number][]
            ).map(([w, o], i) => (
              <div key={i} className="flex items-center gap-1">
                <div
                  className="shrink-0 rounded-[1px]"
                  style={{ width: 8, height: 8, background: v.muted }}
                />
                <div
                  className="rounded-[1px]"
                  style={{
                    width: `${w}%`,
                    height: 3,
                    background: v.mutedFg,
                    opacity: o,
                  }}
                />
              </div>
            ))}
            <div className="mt-auto flex gap-0.5">
              {[v.install, v.update, v.files, v.uninstall].map((c, i) => (
                <div
                  key={i}
                  className="rounded-[1px]"
                  style={{ flex: 1, height: 4, background: c, opacity: 0.75 }}
                />
              ))}
            </div>
          </div>

          {/* Right card */}
          <div
            className="flex flex-[2] flex-col gap-[5%] rounded-[3px] p-[5%]"
            style={{ background: v.card, border: `1px solid ${v.border}` }}
          >
            <div
              className="rounded-[1px]"
              style={{ width: '65%', height: 3, background: v.primary }}
            />
            <div
              className="flex-1 rounded-[2px]"
              style={{ background: v.accent }}
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
            <ThemePreview theme={option.value} />
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
