import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { types } from '../../wailsjs/go/models';
import { mergeVersionDownloads, withZeroDownloads } from './version-downloads';

function version(version: string, downloads = 123): types.VersionInfo {
  return new types.VersionInfo({
    version,
    name: version,
    changelog: '',
    date: '',
    download_url: '',
    game_version: '',
    sha256: '',
    downloads,
    manifest: '',
    prerelease: false,
  });
}

describe('version download merge helpers', () => {
  beforeEach(() => {
    vi.spyOn(console, 'warn').mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('overrides downloads from counts response', () => {
    const result = mergeVersionDownloads(
      [version('1.0.0'), version('1.1.0')],
      { '1.0.0': 50, '1.1.0': 75 },
      'map:map-a',
    );

    expect(result[0].downloads).toBe(50);
    expect(result[1].downloads).toBe(75);
    expect(console.warn).not.toHaveBeenCalled();
  });

  it('defaults missing version counts to zero and warns', () => {
    const result = mergeVersionDownloads(
      [version('1.0.0'), version('1.1.0')],
      { '1.0.0': 10 },
      'mod:mod-a',
    );

    expect(result[0].downloads).toBe(10);
    expect(result[1].downloads).toBe(0);
    expect(console.warn).toHaveBeenCalledWith(
      '[mod:mod-a] Missing download count for version "1.1.0"',
    );
  });

  it('warns when counts include extra unknown versions', () => {
    mergeVersionDownloads(
      [version('1.0.0')],
      { '1.0.0': 3, '9.9.9': 999 },
      'map:map-a',
    );

    expect(console.warn).toHaveBeenCalledWith(
      '[map:map-a] Download counts contain unknown version "9.9.9"',
    );
  });

  it('withZeroDownloads resets all counts to zero', () => {
    const result = withZeroDownloads([
      version('1.0.0', 55),
      version('1.1.0', 77),
    ]);
    expect(result[0].downloads).toBe(0);
    expect(result[1].downloads).toBe(0);
  });
});
