import { useEffect, useState } from 'react';

import { type AssetType, assetTypeToListingPath } from '@/lib/asset-types';

import { GetGalleryImageResponse } from '../../wailsjs/go/registry/Registry';

interface GalleryImageCacheEntry {
  imageUrl: string | null;
  error: boolean;
}

const galleryImageCache = new Map<string, GalleryImageCacheEntry>();
const galleryImageRequests = new Map<string, Promise<GalleryImageCacheEntry>>();

function getCacheKey(type: AssetType, id: string, imagePath: string) {
  return `${type}:${id}:${imagePath}`;
}

async function preloadBrowserImage(imageUrl: string): Promise<void> {
  await new Promise<void>((resolve) => {
    const img = new Image();
    img.decoding = 'async';
    img.onload = () => resolve();
    img.onerror = () => resolve();
    img.src = imageUrl;
  });
}

async function requestGalleryImage(
  type: AssetType,
  id: string,
  imagePath: string,
): Promise<GalleryImageCacheEntry> {
  try {
    const response = await GetGalleryImageResponse(
      assetTypeToListingPath(type),
      id,
      imagePath,
    );
    if (response.status !== 'success') {
      return { imageUrl: null, error: true };
    }

    const imageUrl = response.imageUrl || null;
    if (imageUrl) {
      await preloadBrowserImage(imageUrl);
    }
    return { imageUrl, error: false };
  } catch {
    return { imageUrl: null, error: true };
  }
}

function getOrRequestGalleryImage(
  type: AssetType,
  id: string,
  imagePath: string,
): Promise<GalleryImageCacheEntry> {
  const cacheKey = getCacheKey(type, id, imagePath);
  const cached = galleryImageCache.get(cacheKey);
  if (cached) {
    return Promise.resolve(cached);
  }

  const inFlight = galleryImageRequests.get(cacheKey);
  if (inFlight) {
    return inFlight;
  }

  const request = (async () => {
    try {
      const entry = await requestGalleryImage(type, id, imagePath);
      galleryImageCache.set(cacheKey, entry);
      return entry;
    } finally {
      galleryImageRequests.delete(cacheKey);
    }
  })();

  galleryImageRequests.set(cacheKey, request);
  return request;
}

export async function preloadGalleryImage(
  type: AssetType,
  id: string,
  imagePath?: string,
): Promise<void> {
  if (!imagePath) {
    return;
  }
  await getOrRequestGalleryImage(type, id, imagePath);
}

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
      setImageUrl(null);
      setError(false);
      setLoading(false);
      return;
    }

    let cancelled = false;
    const cacheKey = getCacheKey(type, id, imagePath);
    const cached = galleryImageCache.get(cacheKey);
    if (cached) {
      setImageUrl(cached.imageUrl);
      setError(cached.error);
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(false);

    const loadImage = async () => {
      try {
        const entry = await getOrRequestGalleryImage(type, id, imagePath);
        if (!cancelled) {
          setImageUrl(entry.imageUrl);
          setError(entry.error);
          setLoading(false);
        }
      } catch {
        if (!cancelled) {
          setImageUrl(null);
          setError(true);
          setLoading(false);
        }
      }
    };
    loadImage();

    return () => {
      cancelled = true;
    };
  }, [type, id, imagePath]);

  return { imageUrl, loading, error };
}
