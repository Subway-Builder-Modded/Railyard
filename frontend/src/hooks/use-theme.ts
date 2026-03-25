import { useEffect } from 'react';

import { useProfileStore } from '@/stores/profile-store';

type FullTheme =
  | 'dark'
  | 'light'
  | 'system'
  | 'dark_low'
  | 'dark_high'
  | 'light_low'
  | 'light_high';

const VALID_THEMES = new Set<FullTheme>([
  'dark',
  'light',
  'system',
  'dark_low',
  'dark_high',
  'light_low',
  'light_high',
]);

function normalizeTheme(theme: string): FullTheme {
  if (VALID_THEMES.has(theme as FullTheme)) return theme as FullTheme;
  const lowered = theme.toLowerCase();
  if (lowered.startsWith('dark')) return 'dark';
  if (lowered.startsWith('light')) return 'light';
  return 'system';
}

function applyThemeClasses(root: HTMLElement, theme: Exclude<FullTheme, 'system'>) {
  const isDark = theme === 'dark' || theme === 'dark_low' || theme === 'dark_high';
  root.classList.toggle('dark', isDark);
  root.classList.toggle('soft-light', theme === 'light_low');
  root.classList.toggle('soft-dark', theme === 'dark_low');
  root.classList.toggle('hc-light', theme === 'light_high');
  root.classList.toggle('hc-dark', theme === 'dark_high');
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
