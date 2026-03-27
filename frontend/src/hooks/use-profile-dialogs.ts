import { useCallback, useMemo, useState } from 'react';

import type { types } from '../../wailsjs/go/models';

type ProfilesState = {
  create: {
    open: boolean;
    name: string;
    loading: boolean;
  };
  rename: {
    target: types.UserProfile | null;
    name: string;
    loading: boolean;
  };
  remove: {
    target: types.UserProfile | null;
    loading: boolean;
  };
  swap: {
    target: types.UserProfile | null;
    loading: boolean;
    archiveWarningOpen: boolean;
  };
};

const INITIAL_PROFILES_STATE: ProfilesState = {
  create: {
    open: false,
    name: '',
    loading: false,
  },
  rename: {
    target: null,
    name: '',
    loading: false,
  },
  remove: {
    target: null,
    loading: false,
  },
  swap: {
    target: null,
    loading: false,
    archiveWarningOpen: false,
  },
};

export function useProfilesState() {
  const [dialogs, setDialogs] = useState(INITIAL_PROFILES_STATE);

  // update a specific state section by merging the existing state with the provided partial update.
  const updateProfilesState = useCallback(
    <K extends keyof ProfilesState>(
      key: K,
      patch: Partial<ProfilesState[K]>,
    ) => {
      setDialogs((state) => ({
        ...state,
        [key]: {
          ...state[key],
          ...patch,
        },
      }));
    },
    [],
  );

  // resetState resets a specific state section to its initial state (e.g. when closing a dialog).
  const resetProfilesState = useCallback(
    <K extends keyof ProfilesState>(key: K) => {
      setDialogs((state) => ({
        ...state,
        [key]: INITIAL_PROFILES_STATE[key],
      }));
    },
    [],
  );

  const create = useMemo(
    () => ({
      open: () => updateProfilesState('create', { open: true }),
      setOpen: (open: boolean) => updateProfilesState('create', { open }),
      setName: (name: string) => updateProfilesState('create', { name }),
      setLoading: (loading: boolean) =>
        updateProfilesState('create', { loading }),
      reset: () => resetProfilesState('create'),
    }),
    [updateProfilesState, resetProfilesState],
  );

  const rename = useMemo(
    () => ({
      open: (target: types.UserProfile) =>
        setDialogs((state) => ({
          ...state,
          rename: {
            target,
            name: target.name,
            loading: false,
          },
        })),
      close: () => resetProfilesState('rename'),
      setName: (name: string) => updateProfilesState('rename', { name }),
      setLoading: (loading: boolean) =>
        updateProfilesState('rename', { loading }),
    }),
    [updateProfilesState, resetProfilesState],
  );

  const remove = useMemo(
    () => ({
      open: (target: types.UserProfile) =>
        setDialogs((state) => ({
          ...state,
          remove: {
            target,
            loading: false,
          },
        })),
      close: () => resetProfilesState('remove'),
      setLoading: (loading: boolean) =>
        updateProfilesState('remove', { loading }),
    }),
    [updateProfilesState, resetProfilesState],
  );

  const swap = useMemo(
    () => ({
      open: (target: types.UserProfile) =>
        setDialogs((state) => ({
          ...state,
          swap: {
            target,
            loading: false,
            archiveWarningOpen: false,
          },
        })),
      close: () => resetProfilesState('swap'),
      setLoading: (loading: boolean) =>
        updateProfilesState('swap', { loading }),
      setArchiveWarningOpen: (open: boolean) =>
        updateProfilesState('swap', { archiveWarningOpen: open }),
    }),
    [updateProfilesState, resetProfilesState],
  );

  return { dialogs, create, rename, remove, swap };
}
