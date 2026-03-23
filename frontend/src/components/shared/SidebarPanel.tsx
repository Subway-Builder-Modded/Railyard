import { ChevronRight, SlidersHorizontal } from 'lucide-react';
import {
  type CSSProperties,
  type ReactNode,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from 'react';

import { cn } from '@/lib/utils';
import type { AssetQueryFilters } from '@/stores/asset-query-filter-store';

const SIDEBAR_WIDTH_REM = 15.5;
const SIDEBAR_GAP_REM = 1.5;
const EDGE_GAP_PX = 24;

export const SIDEBAR_CONTENT_OFFSET = `${SIDEBAR_WIDTH_REM + SIDEBAR_GAP_REM}rem`;

function getNavbarOffsetPx(): number {
  return (
    parseFloat(
      getComputedStyle(document.documentElement).getPropertyValue('--app-navbar-offset'),
    ) - 48 || 72
  );
}

export interface SidebarPanelProps {
  open: boolean;
  onToggle: () => void;
  ariaLabel: string;
  filters: AssetQueryFilters;
  children: ReactNode;
  collapsedContent?: ReactNode;
}

export function SidebarPanel({
  open,
  onToggle,
  ariaLabel,
  filters,
  children,
  collapsedContent,
}: SidebarPanelProps) {
  const panelRef = useRef<HTMLElement>(null);
  const toggleRef = useRef<HTMLDivElement>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const [left, setLeft] = useState(0);
  const [showScrollThumb, setShowScrollThumb] = useState(false);
  const [thumbHeight, setThumbHeight] = useState(0);
  const [thumbTop, setThumbTop] = useState(0);

  useLayoutEffect(() => {
    const rootEl = document.getElementById('root');
    const mainEl = document.querySelector<HTMLElement>('main');
    const footerEl = document.querySelector<HTMLElement>('footer');

    let idealTop = getNavbarOffsetPx() + EDGE_GAP_PX;

    const measureLayout = () => {
      idealTop = getNavbarOffsetPx() + EDGE_GAP_PX;
      const maxH = window.innerHeight - idealTop - EDGE_GAP_PX;
      const panel = panelRef.current;
      if (panel) panel.style.maxHeight = `${maxH}px`;
    };

    const updateLeft = () => {
      if (!mainEl) return;
      const { left: l } = mainEl.getBoundingClientRect();
      setLeft(l + (parseFloat(getComputedStyle(mainEl).paddingLeft) || 0));
    };

    const updatePosition = () => {
      const panel = panelRef.current;
      const toggle = toggleRef.current;
      if (!panel) return;

      const footerTop = footerEl ? footerEl.getBoundingClientRect().top : Infinity;
      const panelH = panel.offsetHeight;
      const top = Math.min(idealTop, footerTop - panelH - EDGE_GAP_PX);

      panel.style.top = `${top}px`;
      if (toggle) toggle.style.top = `${idealTop}px`;
    };

    measureLayout();
    updateLeft();
    updatePosition();

    const panelRo = new ResizeObserver(() => updatePosition());
    if (panelRef.current) panelRo.observe(panelRef.current);

    const mainRo = new ResizeObserver(() => {
      updateLeft();
      measureLayout();
      updatePosition();
    });
    if (mainEl) mainRo.observe(mainEl);

    const footerRo = new ResizeObserver(() => {
      measureLayout();
      updatePosition();
    });
    if (footerEl) footerRo.observe(footerEl);

    const handleResize = () => {
      measureLayout();
      updateLeft();
      updatePosition();
    };

    window.addEventListener('resize', handleResize);
    rootEl?.addEventListener('scroll', updatePosition, { passive: true });

    return () => {
      panelRo.disconnect();
      mainRo.disconnect();
      footerRo.disconnect();
      window.removeEventListener('resize', handleResize);
      rootEl?.removeEventListener('scroll', updatePosition);
    };
  }, []);

  const lastFiltersRef = useRef<AssetQueryFilters | null>(null);
  useEffect(() => {
    if (!open) return;
    if (!lastFiltersRef.current) {
      lastFiltersRef.current = filters;
      return;
    }
    document.getElementById('root')?.scrollTo({ top: 0, behavior: 'auto' });
    lastFiltersRef.current = filters;
  }, [filters, open]);

  useLayoutEffect(() => {
    const scrollEl = scrollRef.current;
    if (!scrollEl || !open) {
      setShowScrollThumb(false);
      return;
    }

    const updateThumb = () => {
      const { clientHeight, scrollHeight, scrollTop } = scrollEl;
      const overflow = scrollHeight - clientHeight;
      if (overflow <= 1) {
        setShowScrollThumb(false);
        setThumbHeight(0);
        setThumbTop(0);
        return;
      }
      const nextH = Math.max(24, (clientHeight * clientHeight) / scrollHeight);
      const maxTop = clientHeight - nextH;
      setShowScrollThumb(true);
      setThumbHeight(nextH);
      setThumbTop((scrollTop / overflow) * maxTop);
    };

    updateThumb();
    scrollEl.addEventListener('scroll', updateThumb, { passive: true });
    window.addEventListener('resize', updateThumb);

    const ro = new ResizeObserver(updateThumb);
    ro.observe(scrollEl);
    const contentEl = scrollEl.firstElementChild as HTMLElement | null;
    if (contentEl) ro.observe(contentEl);

    return () => {
      scrollEl.removeEventListener('scroll', updateThumb);
      window.removeEventListener('resize', updateThumb);
      ro.disconnect();
    };
  }, [filters, open]);

  const panelStyle = { position: 'fixed' as const, left, width: `${SIDEBAR_WIDTH_REM}rem` };
  const toggleStyle = { position: 'fixed' as const, left, width: '2.5rem' };

  return (
    <>
      <aside
        ref={panelRef}
        aria-label={ariaLabel}
        className={cn(
          'z-40 flex flex-col overflow-hidden',
          'rounded-2xl border border-border/70 bg-background/90 backdrop-blur-md shadow-sm',
          'transition-[opacity,transform] duration-200 ease-out',
          open
            ? 'opacity-100 translate-x-0 pointer-events-auto'
            : 'opacity-0 -translate-x-3 pointer-events-none',
        )}
        style={panelStyle}
      >
        <div className="flex shrink-0 items-center gap-2 border-b border-border/60 px-[clamp(0.65rem,1.4vw,1rem)] py-[clamp(0.42rem,0.88vw,0.6rem)]">
          <SlidersHorizontal className="h-4 w-4 shrink-0 text-muted-foreground" />
          <span className="flex-1 text-[clamp(0.78rem,0.92vw,0.88rem)] font-semibold text-muted-foreground">
            Filters
          </span>
          <button
            type="button"
            onClick={onToggle}
            aria-label="Collapse filters sidebar"
            className="mr-[-0.15rem] translate-x-0.5 rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-accent/45 hover:text-primary"
          >
            <ChevronRight className="h-4 w-4 rotate-180" />
          </button>
        </div>

        <div className="group/sidebar relative flex min-h-0 flex-1 flex-col">
          <div
            ref={scrollRef}
            className="sidebar-scroll min-h-0 flex-1 overflow-y-auto overflow-x-visible px-[clamp(0.65rem,1.4vw,1rem)] py-3"
            onWheelCapture={(e) => e.stopPropagation()}
          >
            {children}
          </div>

          {showScrollThumb && (
            <div className="pointer-events-none absolute bottom-3 right-1 top-3 w-1 opacity-0 transition-opacity duration-150 group-hover/sidebar:opacity-100">
              <div
                className="absolute left-0 w-full rounded-full bg-[color-mix(in_srgb,var(--foreground)_28%,transparent)]"
                style={
                  { height: thumbHeight, transform: `translateY(${thumbTop}px)` } as CSSProperties
                }
              />
            </div>
          )}
        </div>
      </aside>

      <div
        ref={toggleRef}
        className={cn(
          'z-40 flex flex-col items-stretch overflow-hidden',
          'rounded-xl border border-border/70 bg-background/90 backdrop-blur-md shadow-sm',
          'text-muted-foreground transition-all duration-200 ease-out',
          open
            ? 'opacity-0 pointer-events-none scale-90'
            : 'opacity-100 scale-100 pointer-events-auto',
        )}
        style={toggleStyle}
      >
        <button
          type="button"
          onClick={onToggle}
          aria-label="Expand filters sidebar"
          className="flex h-10 w-full items-center justify-center transition-colors hover:bg-accent/45 hover:text-primary"
        >
          <SlidersHorizontal className="h-4 w-4" />
        </button>

        {collapsedContent && (
          <>
            <div className="h-px w-full shrink-0 bg-border/60" aria-hidden />
            {collapsedContent}
          </>
        )}
      </div>
    </>
  );
}
