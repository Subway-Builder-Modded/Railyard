export type LocalAccentTone =
  | 'install'
  | 'uninstall'
  | 'update'
  | 'import'
  | 'files'
  | 'profiles';

type LocalAccentVariantClasses = {
  solidButton: string;
  outlineButton: string;
  iconButton: string;
  dialogCancel: string;
  dialogPanel: string;
};

const SOLID_TONE_CLASS =
  '!bg-[var(--local-tone-primary)] !text-[var(--local-tone-foreground)] hover:!brightness-90 hover:!text-[var(--local-tone-foreground)]';
const OUTLINE_TONE_CLASS =
  'border-[var(--local-tone-primary)] text-[var(--local-tone-primary)] hover:!bg-[color-mix(in_srgb,var(--local-tone-primary)_20%,transparent)] hover:!text-[var(--local-tone-primary)]';
const ICON_TONE_CLASS =
  'text-[var(--local-tone-primary)] hover:!bg-[color-mix(in_srgb,var(--local-tone-primary)_20%,transparent)] hover:!text-[var(--local-tone-primary)]';
const DIALOG_CANCEL_TONE_CLASS =
  'border-[color-mix(in_srgb,var(--local-tone-primary)_45%,transparent)] text-[var(--local-tone-primary)] hover:!bg-[color-mix(in_srgb,var(--local-tone-primary)_20%,transparent)] hover:!text-[var(--local-tone-primary)]';
const DIALOG_PANEL_TONE_CLASS =
  'border-[color-mix(in_srgb,var(--local-tone-primary)_45%,transparent)] bg-[color-mix(in_srgb,var(--local-tone-primary)_12%,transparent)]';

function buildToneClasses(toneVarsClass: string): LocalAccentVariantClasses {
  const withToneVars = (className: string) => `${toneVarsClass} ${className}`;

  return {
    solidButton: withToneVars(SOLID_TONE_CLASS),
    outlineButton: withToneVars(OUTLINE_TONE_CLASS),
    iconButton: withToneVars(ICON_TONE_CLASS),
    dialogCancel: withToneVars(DIALOG_CANCEL_TONE_CLASS),
    dialogPanel: withToneVars(DIALOG_PANEL_TONE_CLASS),
  };
}

function buildLocalAccentToneClasses<TTone extends string>(toneVarClasses: {
  [Tone in TTone]: string;
}): {
  [Tone in TTone]: LocalAccentVariantClasses;
} {
  const entries = Object.entries(toneVarClasses) as [TTone, string][];

  return Object.fromEntries(
    entries.map(([tone, toneVarsClass]) => [
      tone,
      buildToneClasses(toneVarsClass),
    ]),
  ) as {
    [Tone in TTone]: LocalAccentVariantClasses;
  };
}

const LOCAL_ACCENT_TONE_VARS = {
  install:
    '[--local-tone-primary:var(--install-primary)] [--local-tone-foreground:var(--install-foreground)]',
  uninstall:
    '[--local-tone-primary:var(--uninstall-primary)] [--local-tone-foreground:var(--uninstall-foreground)]',
  update:
    '[--local-tone-primary:var(--update-primary)] [--local-tone-foreground:var(--update-foreground)]',
  import:
    '[--local-tone-primary:var(--import-primary)] [--local-tone-foreground:var(--import-foreground)]',
  files:
    '[--local-tone-primary:var(--files-primary)] [--local-tone-foreground:var(--files-foreground)]',
  profiles:
    '[--local-tone-primary:var(--profiles-primary)] [--local-tone-foreground:var(--profiles-foreground)]',
} satisfies Record<LocalAccentTone, string>;

const LOCAL_ACCENT_TONE_CLASSES = buildLocalAccentToneClasses(
  LOCAL_ACCENT_TONE_VARS,
);

export function getLocalAccentClasses(tone: LocalAccentTone) {
  return LOCAL_ACCENT_TONE_CLASSES[tone];
}

export function getToneVarsClass(tone: LocalAccentTone): string {
  return LOCAL_ACCENT_TONE_VARS[tone];
}

export const LOCAL_ACCENTS = LOCAL_ACCENT_TONE_CLASSES;
