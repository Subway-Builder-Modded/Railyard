import React from 'react';

import { AppFooter } from '@/components/layout/AppFooter';
import {
  APP_CONTENT_SPACING_CLASS,
  APP_SHELL_PADDING_CLASS,
  APP_SHELL_WIDTH_CLASS,
} from '@/components/layout/layout-shell';
import { cn } from '@/lib/utils';

import { GetCurrentVersion } from '../../../wailsjs/go/main/App';
import { Navbar } from './Navbar';

export function Layout({ children }: { children: React.ReactNode }) {
  const [version, setVersion] = React.useState<string>('');

  React.useEffect(() => {
    GetCurrentVersion().then((response) => {
      if (response.status !== 'success') {
        return;
      }
      const sanitized = [...(response.version || '')]
        .filter((c) => c !== '\u0000')
        .join('');
      setVersion(sanitized);
    });
  }, []);
  return (
    <div className="min-h-screen bg-background flex flex-col">
      <Navbar />
      <main
        style={{ paddingTop: 'var(--app-navbar-offset, 6rem)' }}
        className={cn(
          'flex-1',
          APP_SHELL_WIDTH_CLASS,
          APP_SHELL_PADDING_CLASS,
          APP_CONTENT_SPACING_CLASS,
        )}
      >
        {children}
      </main>
      <AppFooter version={version} />
    </div>
  );
}
