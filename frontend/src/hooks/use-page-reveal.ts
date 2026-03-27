import { useEffect, useState } from 'react';

export function usePageReveal(isLoading: boolean) {
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (isLoading) {
      setReady(false);
      return;
    }

    let cancelled = false;
    let secondFrame: number | null = null;
    const firstFrame = window.requestAnimationFrame(() => {
      secondFrame = window.requestAnimationFrame(() => {
        if (!cancelled) {
          setReady(true);
        }
      });
    });

    return () => {
      cancelled = true;
      window.cancelAnimationFrame(firstFrame);
      if (secondFrame !== null) {
        window.cancelAnimationFrame(secondFrame);
      }
    };
  }, [isLoading]);

  return ready;
}
