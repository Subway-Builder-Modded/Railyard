import { useState, useEffect } from "react";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { GetGalleryImage } from "../../../wailsjs/go/registry/Registry";

interface ProjectHeroProps {
  type: "mods" | "maps";
  id: string;
  gallery: string[];
}

export function ProjectHero({ type, id, gallery }: ProjectHeroProps) {
  const [images, setImages] = useState<(string | null)[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null);

  useEffect(() => {
    if (!gallery || gallery.length === 0) {
      setLoading(false);
      return;
    }

    Promise.all(
      gallery.map((path) =>
        GetGalleryImage(type, id, path).catch(() => null)
      )
    ).then((urls) => {
      setImages(urls);
      setLoading(false);
    });
  }, [type, id, gallery]);

  if (loading) {
    return (
      <div className="flex gap-2 overflow-hidden">
        {Array.from({ length: Math.min(gallery?.length || 3, 5) }).map((_, i) => (
          <Skeleton key={i} className="h-24 w-40 rounded-md flex-shrink-0" />
        ))}
      </div>
    );
  }

  const validImages = images.filter((url): url is string => url !== null);

  if (validImages.length === 0) {
    return null;
  }

  return (
    <>
      <div className="flex gap-2 overflow-x-auto pb-1">
        {validImages.map((url, i) => (
          <button
            key={i}
            onClick={() => setSelectedIndex(i)}
            className="h-24 w-40 flex-shrink-0 rounded-md overflow-hidden ring-offset-background transition-opacity hover:opacity-80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          >
            <img
              src={url}
              alt=""
              className="w-full h-full object-cover"
            />
          </button>
        ))}
      </div>

      <Dialog open={selectedIndex !== null} onOpenChange={() => setSelectedIndex(null)}>
        <DialogContent className="w-[95vw] sm:max-w-none max-h-[95vh] p-2 bg-background/95 backdrop-blur-sm border-border">
          {selectedIndex !== null && validImages[selectedIndex] && (
            <div className="relative flex items-center justify-center">
              <img
                src={validImages[selectedIndex]}
                alt=""
                className="max-h-[90vh] rounded-md object-contain"
              />
              {validImages.length > 1 && (
                <>
                  <Button
                    variant="secondary"
                    size="icon"
                    className="absolute left-2 top-1/2 -translate-y-1/2 bg-background/80 backdrop-blur-sm"
                    onClick={() =>
                      setSelectedIndex((selectedIndex - 1 + validImages.length) % validImages.length)
                    }
                  >
                    <ChevronLeft className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="secondary"
                    size="icon"
                    className="absolute right-2 top-1/2 -translate-y-1/2 bg-background/80 backdrop-blur-sm"
                    onClick={() =>
                      setSelectedIndex((selectedIndex + 1) % validImages.length)
                    }
                  >
                    <ChevronRight className="h-4 w-4" />
                  </Button>
                </>
              )}
              <div className="absolute bottom-2 left-1/2 -translate-x-1/2 text-xs text-muted-foreground bg-background/80 backdrop-blur-sm px-2 py-0.5 rounded-full">
                {selectedIndex + 1} / {validImages.length}
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}
