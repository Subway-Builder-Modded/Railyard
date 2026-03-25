import { BookText, Megaphone } from 'lucide-react';

import { DiscordIcon, GitHubIcon } from '@/components/icons/social-icons';
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
    icon: GitHubIcon,
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
            {version || 'Unknown Version'}
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
