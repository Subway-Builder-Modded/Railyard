import {
  Compass,
  Inbox,
  Play,
  RefreshCw,
  Settings,
  Square,
  Terminal,
  TrainTrack,
} from 'lucide-react';
import { type ComponentType, useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import { Link, useLocation } from 'wouter';

import {
  APP_SHELL_PADDING_CLASS,
  APP_SHELL_WIDTH_CLASS,
} from '@/components/layout/layout-shell';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';
import { useConfigStore } from '@/stores/config-store';
import { useGameStore } from '@/stores/game-store';
import { useInstalledStore } from '@/stores/installed-store';
import { useRegistryStore } from '@/stores/registry-store';

type NavLinkConfig = {
  href: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  isCurrent: (location: string) => boolean;
};

const navLinks: NavLinkConfig[] = [
  {
    href: '/library',
    label: 'Library',
    icon: Inbox,
    isCurrent: (location: string) => location.startsWith('/library'),
  },
  {
    href: '/search',
    label: 'Browse',
    icon: Compass,
    isCurrent: (location: string) =>
      location.startsWith('/search') || location.startsWith('/project'),
  },
  {
    href: '/logs',
    label: 'Logs',
    icon: Terminal,
    isCurrent: (location: string) => location.startsWith('/logs'),
  },
  {
    href: '/settings',
    label: 'Settings',
    icon: Settings,
    isCurrent: (location: string) => location.startsWith('/settings'),
  },
] as const;

const MOD_REMINDER_KEY = 'railyard:mod-reminder-acknowledged';
const NAV_ITEM_BASE_CLASS =
  'group relative flex items-center gap-2 rounded-lg px-[clamp(0.45rem,0.95vw,0.7rem)] py-[clamp(0.4rem,0.82vw,0.56rem)] text-[clamp(0.8rem,0.95vw,0.9rem)] font-semibold text-muted-foreground transition-all duration-150';
const NAV_ITEM_GREEN_HOVER_CLASS = 'hover:text-primary hover:bg-accent/45';
const NAV_CURRENT_INDICATOR_CLASS =
  'absolute -bottom-[0.38rem] left-1/2 h-1 w-[calc(100%-1rem)] -translate-x-1/2 rounded-full bg-primary';
const NAVBAR_TOP_OFFSET_PX = 48;
const NAVBAR_BOTTOM_GAP_PX = 12;

export function Navbar() {
  const headerRef = useRef<HTMLElement>(null);
  const [location] = useLocation();
  const { refresh, loading, refreshing } = useRegistryStore();
  const canLaunch = useConfigStore((s) => s.validation?.executablePathValid);
  const { running, launch, stop } = useGameStore();
  const installedMaps = useInstalledStore((s) => s.installedMaps);
  const [showModReminder, setShowModReminder] = useState(false);

  const runWithToast = async (
    action: () => Promise<void>,
    fallbackMessage: string,
  ) => {
    try {
      await action();
    } catch (err) {
      toast.error(String(err) || fallbackMessage);
    }
  };

  const handleLaunch = async () => {
    const hasMaps = installedMaps.length > 0;
    const alreadyAcknowledged =
      localStorage.getItem(MOD_REMINDER_KEY) === 'true';

    if (hasMaps && !alreadyAcknowledged) {
      setShowModReminder(true);
      return;
    }

    await runWithToast(launch, 'Failed to launch game.');
  };

  const handleAcknowledgeAndLaunch = async () => {
    localStorage.setItem(MOD_REMINDER_KEY, 'true');
    setShowModReminder(false);
    await runWithToast(launch, 'Failed to launch game.');
  };

  const handleStop = async () => runWithToast(stop, 'Failed to stop game.');

  useEffect(() => {
    const element = headerRef.current;
    if (!element) {
      return;
    }

    const updateOffset = () => {
      const offset = Math.ceil(
        element.getBoundingClientRect().height +
          NAVBAR_TOP_OFFSET_PX +
          NAVBAR_BOTTOM_GAP_PX,
      );
      document.documentElement.style.setProperty(
        '--app-navbar-offset',
        `${offset}px`,
      );
    };

    updateOffset();
    const observer = new ResizeObserver(updateOffset);
    observer.observe(element);
    window.addEventListener('resize', updateOffset);

    return () => {
      observer.disconnect();
      window.removeEventListener('resize', updateOffset);
      document.documentElement.style.removeProperty('--app-navbar-offset');
    };
  }, []);

  return (
    <header ref={headerRef} className="fixed inset-x-0 top-3 z-50">
      <div className={cn(APP_SHELL_WIDTH_CLASS, APP_SHELL_PADDING_CLASS)}>
        <div className="flex min-h-[4rem] flex-wrap items-center justify-between gap-y-2 rounded-2xl border border-border/70 bg-background/90 px-[clamp(0.8rem,2vw,1.4rem)] py-1.5 shadow-sm backdrop-blur-md">
          <div className="flex min-w-0 flex-wrap items-center gap-[clamp(0.6rem,1.8vw,1.25rem)]">
            <Link
              href="/"
              className="flex items-center gap-2.5 rounded-md px-2 py-1.5 text-[clamp(1rem,1.55vw,1.15rem)] font-extrabold tracking-[0.01em] text-foreground"
            >
              <TrainTrack className="h-[1.2em] w-[1.2em]" />
              <span>Railyard</span>
            </Link>
            <nav className="flex max-w-full flex-wrap items-center gap-1.5">
              {navLinks.map(({ href, label, icon: Icon, isCurrent }) => {
                const current = isCurrent(location);

                return (
                  <Link
                    key={href}
                    href={href}
                    aria-current={current ? 'page' : undefined}
                    className={cn(
                      NAV_ITEM_BASE_CLASS,
                      NAV_ITEM_GREEN_HOVER_CLASS,
                      current ? 'text-primary bg-accent/45' : undefined,
                    )}
                  >
                    <Icon className="h-[1.05em] w-[1.05em] shrink-0 transition-colors" />
                    <span>{label}</span>
                    {current && (
                      <span
                        aria-hidden
                        className={NAV_CURRENT_INDICATOR_CLASS}
                      />
                    )}
                  </Link>
                );
              })}
            </nav>
          </div>
          <div className="flex items-center gap-1.5">
            {running ? (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleStop}
                className={cn(
                  NAV_ITEM_BASE_CLASS,
                  'h-auto bg-[color-mix(in_srgb,var(--install-primary)_20%,transparent)] text-[var(--install-primary)] hover:!bg-[color-mix(in_srgb,var(--uninstall-primary)_24%,transparent)] hover:!text-[var(--uninstall-primary)]',
                )}
              >
                <Square className="mr-1.5 h-[1.125rem] w-[1.125rem]" />
                Running
                <span
                  aria-hidden
                  className={cn(
                    NAV_CURRENT_INDICATOR_CLASS,
                    'bg-[var(--install-primary)] transition-colors group-hover:bg-[var(--uninstall-primary)]',
                  )}
                />
              </Button>
            ) : (
              <Tooltip>
                <TooltipTrigger asChild>
                  <span>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleLaunch}
                      disabled={!canLaunch}
                      className={cn(
                        NAV_ITEM_BASE_CLASS,
                        NAV_ITEM_GREEN_HOVER_CLASS,
                        'h-auto disabled:opacity-50',
                      )}
                    >
                      <Play className="mr-1.5 h-[1.125rem] w-[1.125rem]" />
                      Launch
                    </Button>
                  </span>
                </TooltipTrigger>
                {!canLaunch && (
                  <TooltipContent>
                    Configure game executable in Settings first
                  </TooltipContent>
                )}
              </Tooltip>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={refresh}
              disabled={loading || refreshing}
              className={cn(
                NAV_ITEM_BASE_CLASS,
                NAV_ITEM_GREEN_HOVER_CLASS,
                'h-auto',
              )}
            >
              <RefreshCw
                className={cn(
                  'mr-1 h-[1.125rem] w-[1.125rem]',
                  (loading || refreshing) && 'animate-spin',
                )}
              />
              Refresh
            </Button>
          </div>
        </div>
      </div>

      <Dialog open={showModReminder} onOpenChange={setShowModReminder}>
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>Enable Railyard Map Loader</DialogTitle>
            <DialogDescription>
              You have custom maps installed. To use them in-game, make sure the{' '}
              <span className="font-semibold text-foreground">
                Railyard Map Loader
              </span>{' '}
              mod is enabled in the game's mod manager.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowModReminder(false)}>
              Cancel
            </Button>
            <Button onClick={handleAcknowledgeAndLaunch}>
              Got it, launch game
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </header>
  );
}
