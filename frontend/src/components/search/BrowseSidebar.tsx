import { ChevronRight, SlidersHorizontal } from 'lucide-react';
import {
  type CSSProperties,
  type Dispatch,
  type SetStateAction,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from 'react';

import { SidebarFilters } from '@/components/search/SidebarFilters';
import type { AssetType } from '@/lib/asset-types';
import { cn } from '@/lib/utils';
import type { SearchFilterState } from '@/stores/search-store';

/** Width of the expanded sidebar panel (rem). */
const SIDEBAR_WIDTH_REM = 15.5;

/** Gap between sidebar right edge and page content (rem). */
const SIDEBAR_GAP_REM = 1.5;

/** Symmetric vertical gap above/below the sidebar within the viewport (px). */
const VIEWPORT_EDGE_GAP_PX = 24;

/**
 * CSS padding-left to apply to page content when the sidebar is open,
 * so content does not overlap the floating panel.
 */
export const SIDEBAR_CONTENT_OFFSET = `${SIDEBAR_WIDTH_REM + SIDEBAR_GAP_REM}rem`;

export interface BrowseSidebarProps {
  open: boolean;
  onToggle: () => void;
  filters: SearchFilterState;
  onFiltersChange: Dispatch<SetStateAction<SearchFilterState>>;
  onTypeChange: (type: AssetType) => void;
  availableTags: string[];
  availableSpecialDemand: string[];
  modTagCounts: Record<string, number>;
  mapLocationCounts: Record<string, number>;
  mapSourceQualityCounts: Record<string, number>;
  mapLevelOfDetailCounts: Record<string, number>;
  mapSpecialDemandCounts: Record<string, number>;
  modCount: number;
  mapCount: number;
}

function getNavbarOffsetPx(): number {
  return (
    parseFloat(
      getComputedStyle(document.documentElement).getPropertyValue('--app-navbar-offset'),
    ) || 72
  );
}

function getMainContentLeft(): number {
  const el = document.querySelector<HTMLElement>('main');
  if (!el) return 16;
  const { left } = el.getBoundingClientRect();
  return left + (parseFloat(getComputedStyle(el).paddingLeft) || 0);
}

/** `top` value (px) pinned below the navbar by a constant viewport gap. */
function computeSidebarTop(): number {
  return getNavbarOffsetPx() - 24;
}

/**
 * Floating filter sidebar for the Browse page.
 *
 * Vertical position:
 *   The sidebar starts at a stable top offset under the navbar and grows
 *   downward as content increases. `maxHeight` is recalculated with viewport
 *   size so top/bottom viewport gaps stay symmetric.
 *
 * Position mode:
 *   • `fixed`    — default; sidebar is anchored to the viewport.
 *   • `absolute` — engaged when the footer enters the viewport AND the sidebar
 *                  is tall enough to actually reach it.  The absolute `top` is
 *                  derived deterministically from the footer's document-space
 *                  position and the sidebar height, so the breakpoint is
 *                  scroll-speed-independent.  The sidebar then scrolls with
 *                  the page. Reverts to `fixed` when the footer leaves
 *                  the viewport.
 */
export function BrowseSidebar({ open, onToggle, ...filterProps }: BrowseSidebarProps) {
  const panelRef = useRef<HTMLElement>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const [left, setLeft] = useState(0);
  const [anchored, setAnchored] = useState(false);
  const [viewportHeight, setViewportHeight] = useState(
    typeof window !== 'undefined' ? window.innerHeight : 800,
  );
  const [showScrollThumb, setShowScrollThumb] = useState(false);
  const [thumbHeight, setThumbHeight] = useState(0);
  const [thumbTop, setThumbTop] = useState(0);
  const anchoredTopRef = useRef(0);

  useLayoutEffect(() => {
    const updateLeft = () => setLeft(getMainContentLeft());
    const updateViewportHeight = () => setViewportHeight(window.innerHeight);
    const recomputeAnchor = () => {
      if (!open) {
        setAnchored(false);
        return;
      }

      const footerEl = document.querySelector<HTMLElement>('footer');
      const sH = panelRef.current?.offsetHeight ?? 0;
      if (!footerEl || sH === 0) {
        setAnchored(false);
        return;
      }

      const fixedTop = computeSidebarTop();
      const footerRect = footerEl.getBoundingClientRect();
      const footerTopViewport = footerRect.top;
      const footerInViewport = footerTopViewport <= window.innerHeight;

      if (!footerInViewport) {
        setAnchored(false);
        return;
      }

      // Guard: only anchor if the sidebar is tall enough to actually reach
      // the footer from its fixed start position.
      if (fixedTop + sH < footerTopViewport - VIEWPORT_EDGE_GAP_PX) {
        setAnchored(false);
        return;
      }

      anchoredTopRef.current =
        footerTopViewport + window.scrollY - sH - VIEWPORT_EDGE_GAP_PX;
      setAnchored(true);
    };
    const updateAll = () => {
      updateLeft();
      updateViewportHeight();
      recomputeAnchor();
    };
    let rafId = 0;
    const scheduleUpdate = () => {
      cancelAnimationFrame(rafId);
      rafId = requestAnimationFrame(updateAll);
    };

    updateAll();

    const mainEl = document.querySelector<HTMLElement>('main');

    // ResizeObserver covers both <main> (left alignment) and the panel itself
    // (height, used for footer-overlap guard calculations).
    const ro = new ResizeObserver((entries) => {
      for (const entry of entries) {
        if (entry.target === mainEl) updateLeft();
      }
      scheduleUpdate();
    });
    if (mainEl) ro.observe(mainEl);
    if (panelRef.current) ro.observe(panelRef.current);
    window.addEventListener('resize', scheduleUpdate);
    window.addEventListener('scroll', scheduleUpdate, { passive: true });

    return () => {
      cancelAnimationFrame(rafId);
      ro.disconnect();
      window.removeEventListener('resize', scheduleUpdate);
      window.removeEventListener('scroll', scheduleUpdate);
    };
  }, [open]);

  const lastFiltersRef = useRef<SearchFilterState | null>(null);
  useEffect(() => {
    if (!open) return;
    if (!lastFiltersRef.current) {
      lastFiltersRef.current = filterProps.filters;
      return;
    }

    // Product behavior: after changing filters, snap to the top of the page.
    window.scrollTo({ top: 0, left: 0, behavior: 'auto' });
    lastFiltersRef.current = filterProps.filters;
  }, [filterProps.filters, open]);

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

      const nextThumbHeight = Math.max(24, (clientHeight * clientHeight) / scrollHeight);
      const maxThumbTop = clientHeight - nextThumbHeight;
      const nextThumbTop = (scrollTop / overflow) * maxThumbTop;

      setShowScrollThumb(true);
      setThumbHeight(nextThumbHeight);
      setThumbTop(nextThumbTop);
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
  }, [filterProps.filters, open]);

  const fixedTop = computeSidebarTop();
  const maxHeight = viewportHeight - fixedTop - VIEWPORT_EDGE_GAP_PX;

  const expandedPositionStyle = open && anchored
    ? ({ position: 'absolute', top: anchoredTopRef.current, left } as const)
    : ({ position: 'fixed', top: fixedTop, left } as const);
  const collapsedPositionStyle = { position: 'fixed', top: fixedTop, left } as const;

  return (
    <>
      {/* ── Expanded panel ─────────────────────────────────────────────── */}
      <aside
        ref={panelRef}
        aria-label="Browse filters"
        className={cn(
          'z-40 flex flex-col overflow-hidden',
          'rounded-2xl border border-border/70 bg-background/90 backdrop-blur-md shadow-sm',
          'transition-[opacity,transform] duration-200 ease-out',
          open
            ? 'opacity-100 translate-x-0 pointer-events-auto'
            : 'opacity-0 -translate-x-3 pointer-events-none',
        )}
        style={{ ...expandedPositionStyle, width: `${SIDEBAR_WIDTH_REM}rem`, maxHeight }}
      >
        {/* Header — proportions mirrored from navbar pill */}
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

        {/* Scrollable filter content with custom overlay scrollbar. */}
        <div className="group/sidebar relative flex min-h-0 flex-1 flex-col">
          <div
            ref={scrollRef}
            className="sidebar-scroll min-h-0 flex-1 overflow-y-auto overflow-x-visible px-[clamp(0.65rem,1.4vw,1rem)] py-3"
            onWheelCapture={(event) => event.stopPropagation()}
          >
            <SidebarFilters {...filterProps} />
          </div>

          {showScrollThumb && (
            <div className="pointer-events-none absolute bottom-3 right-1 top-3 w-1 opacity-0 transition-opacity duration-150 group-hover/sidebar:opacity-100">
              <div
                className="absolute left-0 w-full rounded-full bg-[color-mix(in_srgb,var(--foreground)_28%,transparent)]"
                style={{ height: thumbHeight, transform: `translateY(${thumbTop}px)` } as CSSProperties}
              />
            </div>
          )}
        </div>
      </aside>

      {/* ── Collapsed toggle pill ───────────────────────────────────────── */}
      <button
        type="button"
        onClick={onToggle}
        aria-label="Expand filters sidebar"
        className={cn(
          'z-40 flex items-center justify-center',
          'rounded-xl border border-border/70 bg-background/90 backdrop-blur-md shadow-sm',
          'text-muted-foreground transition-all duration-200 ease-out',
          'hover:bg-accent/45 hover:text-primary',
          open
            ? 'opacity-0 pointer-events-none scale-90'
            : 'opacity-100 scale-100 pointer-events-auto',
        )}
        style={{ ...collapsedPositionStyle, width: '2.5rem', height: '2.5rem' }}
      >
        <SlidersHorizontal className="h-4 w-4" />
      </button>
    </>
  );
}
