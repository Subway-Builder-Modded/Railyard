import {
  AssetSidebarPanel,
  type AssetSidebarPanelProps,
  SIDEBAR_CONTENT_OFFSET,
} from '@/components/shared/AssetSidebarPanel';

export const LIBRARY_SIDEBAR_CONTENT_OFFSET = SIDEBAR_CONTENT_OFFSET;

export type LibrarySidebarPanelProps = Omit<AssetSidebarPanelProps, 'ariaLabel'>;

export function LibrarySidebarPanel(props: LibrarySidebarPanelProps) {
  return <AssetSidebarPanel {...props} ariaLabel="Library filters" />;
}
