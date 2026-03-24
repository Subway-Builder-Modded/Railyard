import { beforeEach, describe, expect, it } from 'vitest';

import { useLibraryStore } from '@/stores/library-store';

describe('useLibraryStore per-asset-type state', () => {
  beforeEach(() => {
    useLibraryStore.setState({
      filters: {
        query: '',
        type: 'mod',
        perPage: 12,
        sort: { field: 'name', direction: 'asc' },
        randomSeed: 200,
        mod: { tags: [] },
        map: {
          locations: [],
          sourceQuality: [],
          levelOfDetail: [],
          specialDemand: [],
        },
      },
      page: 1,
      scopedByType: {
        map: {
          sort: { field: 'name', direction: 'asc' },
          randomSeed: 200,
          mod: { tags: [] },
          map: {
            locations: [],
            sourceQuality: [],
            levelOfDetail: [],
            specialDemand: [],
          },
          page: 1,
        },
        mod: {
          sort: { field: 'name', direction: 'asc' },
          randomSeed: 200,
          mod: { tags: [] },
          map: {
            locations: [],
            sourceQuality: [],
            levelOfDetail: [],
            specialDemand: [],
          },
          page: 1,
        },
      },
      selectedIds: new Set<string>(),
    });
  });

  it('restores map/mod scoped sort, filters, random seed, and page when switching type', () => {
    useLibraryStore.setState((state) => ({
      ...state,
      filters: {
        ...state.filters,
        sort: { field: 'author', direction: 'desc' },
        randomSeed: 31,
        mod: {
          ...state.filters.mod,
          tags: ['gameplay'],
        },
      },
      page: 4,
    }));

    useLibraryStore.getState().setType('map');

    useLibraryStore.setState((state) => ({
      ...state,
      filters: {
        ...state.filters,
        sort: { field: 'population', direction: 'asc' },
        randomSeed: 41,
        map: {
          ...state.filters.map,
          locations: ['north-america'],
        },
      },
      page: 2,
    }));

    useLibraryStore.getState().setType('mod');
    let state = useLibraryStore.getState();
    expect(state.filters.type).toBe('mod');
    expect(state.filters.sort).toEqual({ field: 'author', direction: 'desc' });
    expect(state.filters.randomSeed).toBe(31);
    expect(state.filters.mod.tags).toEqual(['gameplay']);
    expect(state.page).toBe(4);

    useLibraryStore.getState().setType('map');
    state = useLibraryStore.getState();
    expect(state.filters.type).toBe('map');
    expect(state.filters.sort).toEqual({
      field: 'population',
      direction: 'asc',
    });
    expect(state.filters.randomSeed).toBe(41);
    expect(state.filters.map.locations).toEqual(['north-america']);
    expect(state.page).toBe(2);
  });

  it('keeps query and perPage shared across type switches', () => {
    useLibraryStore.getState().setFilters((prev) => ({
      ...prev,
      query: 'routes',
      perPage: 48,
    }));

    useLibraryStore.getState().setType('map');
    let state = useLibraryStore.getState();
    expect(state.filters.query).toBe('routes');
    expect(state.filters.perPage).toBe(48);

    useLibraryStore.getState().setType('mod');
    state = useLibraryStore.getState();
    expect(state.filters.query).toBe('routes');
    expect(state.filters.perPage).toBe(48);
  });

  it('removes only specified selected ids', () => {
    useLibraryStore.getState().selectAll(['mod-a', 'map-b', 'mod-c']);

    useLibraryStore.getState().removeSelected(['map-b', 'missing-id']);

    const state = useLibraryStore.getState();
    expect(Array.from(state.selectedIds).sort()).toEqual(['mod-a', 'mod-c']);
  });
});

describe('useLibraryStore selection', () => {
  beforeEach(() => {
    useLibraryStore.setState({ selectedIds: new Set<string>() });
  });

  it('toggleSelected adds an unselected id', () => {
    useLibraryStore.getState().toggleSelected('mod-a');
    expect(useLibraryStore.getState().selectedIds.has('mod-a')).toBe(true);
  });

  it('toggleSelected removes an already-selected id', () => {
    useLibraryStore.setState({ selectedIds: new Set(['mod-a', 'mod-b']) });
    useLibraryStore.getState().toggleSelected('mod-a');
    expect(useLibraryStore.getState().selectedIds.has('mod-a')).toBe(false);
    expect(useLibraryStore.getState().selectedIds.has('mod-b')).toBe(true);
  });

  it('selectAll replaces the entire selection', () => {
    useLibraryStore.setState({ selectedIds: new Set(['old-id']) });
    useLibraryStore.getState().selectAll(['new-a', 'new-b']);
    expect(Array.from(useLibraryStore.getState().selectedIds).sort()).toEqual([
      'new-a',
      'new-b',
    ]);
  });

  it('clearSelection empties the set', () => {
    useLibraryStore.setState({ selectedIds: new Set(['mod-a', 'map-b']) });
    useLibraryStore.getState().clearSelection();
    expect(useLibraryStore.getState().selectedIds.size).toBe(0);
  });

  it('isSelected returns true for selected ids and false otherwise', () => {
    useLibraryStore.setState({ selectedIds: new Set(['mod-a']) });
    expect(useLibraryStore.getState().isSelected('mod-a')).toBe(true);
    expect(useLibraryStore.getState().isSelected('mod-b')).toBe(false);
  });

  it('removeSelected is a no-op for an empty list', () => {
    useLibraryStore.setState({ selectedIds: new Set(['mod-a']) });
    useLibraryStore.getState().removeSelected([]);
    expect(useLibraryStore.getState().selectedIds.has('mod-a')).toBe(true);
  });
});
