import { Gamepad2, RefreshCw, TrainTrack } from 'lucide-react';
import { type ComponentType, type SVGProps } from 'react';

import { GitHubIcon } from '@/components/icons/social-icons';

import type { types } from '../../../wailsjs/go/models';

export interface SetupStepState {
  validation: types.ConfigPathValidation | null;
  tokenState: 'idle' | 'valid' | 'invalid';
  checkForUpdates: boolean | null;
}

export interface SetupStep {
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  title: string;
  label: string;
  description: string;
  canProceed: (state: SetupStepState) => boolean;
}

export const SETUP_STEPS: SetupStep[] = [
  {
    icon: TrainTrack,
    title: 'Data Folder',
    label: 'Data Folder',
    description: 'Set the path to your metro-maker4 data folder.',
    canProceed: (s) => !!s.validation?.metroMakerDataPathValid,
  },
  {
    icon: Gamepad2,
    title: 'Game Executable',
    label: 'Executable',
    description: 'Set the path to your game executable.',
    canProceed: (s) => !!s.validation?.executablePathValid,
  },
  {
    icon: GitHubIcon,
    title: 'GitHub Token',
    label: 'GitHub',
    description:
      'Enter a token for higher GitHub API rate limits. For more info, see the documentation.',
    canProceed: (s) => s.tokenState !== 'invalid',
  },
  {
    icon: RefreshCw,
    title: 'Automatic Updates',
    label: 'Updates',
    description:
      'Configure whether the app should check for updates automatically on launch.',
    canProceed: (s) => s.checkForUpdates !== null,
  },
];
