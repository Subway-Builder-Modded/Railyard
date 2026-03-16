import { useEffect, useState } from 'react';

import { type AssetType, assetTypeToListingPath } from '@/lib/asset-types';

import { GetGalleryImage } from '../../wailsjs/go/registry/Registry';

export function useGalleryImage(
  type: AssetType,
  id: string,
  imagePath?: string,
) {
  const [imageUrl, setImageUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (!imagePath) {
      setLoading(false);
      return;
    }

    let cancelled = false;
    setLoading(true);
    setError(false);

    GetGalleryImage(assetTypeToListingPath(type), id, imagePath)
      .then((url) => {
        if (!cancelled) {
          setImageUrl(url);
          setLoading(false);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setError(true);
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [type, id, imagePath]);

  return { imageUrl, loading, error };
}
