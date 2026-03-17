import { useEffect } from 'react';

import { useProfileStore } from '@/stores/profile-store';

const DARK_THEMES = new Set(['dark', 'dark_low', 'dark_high']);

const VARIANT_CLASS_MAP: Record<string, string | null> = {
  'dark': null,
  'dark_low': 'theme-dark_low',
  'dark_high': 'theme-dark_high',
  'light': null,
  'light_low': 'theme-light_low',
  'light_high': 'theme-light_high',
  'system': null,
};

const ALL_VARIANT_CLASSES = [
  'theme-dark_low',
  'theme-dark_high',
  'theme-light_low',
  'theme-light_high',
];

function applyThemeClasses(root: HTMLElement, effectiveTheme: string) {
  // Strip all variant classes first
  root.classList.remove(...ALL_VARIANT_CLASSES);
  // Set dark base
  root.classList.toggle('dark', DARK_THEMES.has(effectiveTheme));
  // Apply variant class if needed
  const variantClass = VARIANT_CLASS_MAP[effectiveTheme] ?? null;
  if (variantClass) root.classList.add(variantClass);
}

export function useTheme() {
  const theme = useProfileStore(
    (s) => s.profile?.uiPreferences?.theme ?? 'system',
  );

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
