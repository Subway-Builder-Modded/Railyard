import { useEffect } from 'react';

import { useProfileStore } from '@/stores/profile-store';

type FullTheme =
  | 'dark'
  | 'light'
  | 'system'
  | 'midnight'
  | 'coffee'
  | 'forest'
  | 'crystal';

const VALID_THEMES = new Set<FullTheme>([
  'dark',
  'light',
  'system',
  'midnight',
  'coffee',
  'forest',
  'crystal',
]);

function normalizeTheme(theme: string): FullTheme {
  if (VALID_THEMES.has(theme as FullTheme)) return theme as FullTheme;
  const lowered = theme.toLowerCase();
  if (lowered.startsWith('dark')) return 'dark';
  if (lowered.startsWith('light')) return 'light';
  return 'system';
}

function applyThemeClasses(
  root: HTMLElement,
  theme: Exclude<FullTheme, 'system'>,
) {
  const isDark =
    theme === 'dark' ||
    theme === 'midnight' ||
    theme === 'forest' ||
    theme === 'coffee';
  root.classList.toggle('dark', isDark);
  root.classList.toggle('midnight', theme === 'midnight');
  root.classList.toggle('coffee', theme === 'coffee');
  root.classList.toggle('forest', theme === 'forest');
  root.classList.toggle('crystal', theme === 'crystal');
}

export function useTheme() {
  const rawTheme = useProfileStore((s) => s.profile?.uiPreferences?.theme);
  const theme = normalizeTheme(rawTheme ?? 'system');

  useEffect(() => {
    const root = document.documentElement;

    if (!root.classList.contains('theme-ready')) {
      requestAnimationFrame(() => root.classList.add('theme-ready'));
    }

    if (theme === 'system') {
      const mql = window.matchMedia('(prefers-color-scheme: dark)');
      applyThemeClasses(root, mql.matches ? 'dark' : 'light');

      const handler = (e: MediaQueryListEvent) => {
        applyThemeClasses(root, e.matches ? 'dark' : 'light');
      };
      mql.addEventListener('change', handler);
      return () => mql.removeEventListener('change', handler);
    }

    applyThemeClasses(root, theme);
  }, [theme]);
}
