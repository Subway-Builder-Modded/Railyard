import { BookText, Github, Megaphone } from 'lucide-react';

import {
  APP_SHELL_PADDING_CLASS,
  APP_SHELL_WIDTH_CLASS,
} from '@/components/layout/layout-shell';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime';

const COMMUNITY_LINKS = [
  {
    id: 'docs',
    label: 'Documentation',
    href: 'https://subwaybuildermodded.com/railyard/docs',
    icon: BookText,
  },
  {
    id: 'updates',
    label: 'Updates',
    href: 'https://subwaybuildermodded.com/railyard/updates',
    icon: Megaphone,
  },
  {
    id: 'discord',
    label: 'Discord',
    href: 'https://discord.gg/syG9YHMyeG',
    icon: DiscordIcon,
  },
  {
    id: 'github',
    label: 'GitHub',
    href: 'https://github.com/Subway-Builder-Modded',
    icon: Github,
  },
] as const;

export interface AppFooterProps {
  version: string;
}

export function AppFooter({ version }: AppFooterProps) {
  return (
    <footer className="pb-5 pt-6 sm:pb-7 sm:pt-8">
      <div className={cn(APP_SHELL_WIDTH_CLASS, APP_SHELL_PADDING_CLASS)}>
        <div className="border-t border-border/60 pt-4">
          <p className="text-center text-sm font-medium text-muted-foreground">
            Railyard {version || 'v0.0.0'} | &copy; Subway Builder Modded 2026.
          </p>

          <div className="mt-2.5">
            <div className="flex items-center justify-center gap-2.5">
              {COMMUNITY_LINKS.map(({ id, href, icon: Icon, label }) => (
                <Button
                  key={id}
                  type="button"
                  intent="plain"
                  size="sm"
                  onClick={() => BrowserOpenURL(href)}
                  aria-label={label}
                  className="gap-2 text-muted-foreground hover:text-primary font-semibold"
                >
                  <Icon className="size-5" />
                  <span>{label}</span>
                </Button>
              ))}
            </div>
          </div>
        </div>
      </div>
    </footer>
  );
}

function DiscordIcon({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      aria-hidden="true"
      className={cn('fill-current', className)}
    >
      <path d="M20.317 4.369A19.791 19.791 0 0 0 15.885 3c-.191.337-.404.792-.553 1.149a18.266 18.266 0 0 0-5.665 0A11.437 11.437 0 0 0 9.114 3a19.736 19.736 0 0 0-4.433 1.369C1.88 8.557 1.12 12.642 1.5 16.671a19.927 19.927 0 0 0 5.427 2.745c.438-.617.828-1.27 1.17-1.953a12.912 12.912 0 0 1-1.84-.888c.155-.113.307-.23.454-.352a14.184 14.184 0 0 0 10.578 0 8.464 8.464 0 0 0 .454.352c-.588.347-1.203.644-1.84.888.342.683.732 1.336 1.17 1.953A19.868 19.868 0 0 0 22.5 16.67c.445-4.669-.76-8.716-2.183-12.302ZM8.678 14.209c-1.035 0-1.886-.95-1.886-2.112 0-1.162.831-2.112 1.886-2.112 1.062 0 1.9.958 1.886 2.112 0 1.162-.831 2.112-1.886 2.112Zm6.644 0c-1.035 0-1.886-.95-1.886-2.112 0-1.162.831-2.112 1.886-2.112 1.062 0 1.9.958 1.886 2.112 0 1.162-.824 2.112-1.886 2.112Z" />
    </svg>
  );
}
