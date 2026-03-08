import { useEffect, useRef } from "react";
import { EventsOn } from "../../../wailsjs/runtime/runtime";
import { toast } from "sonner";
import { Download } from "lucide-react";

interface ExtractProgress {
    itemId: string,
    amountExtracted: number,
    total: number,
}

export function ExtractNotification() {
  const toastIds = useRef<Map<string, string | number>>(new Map());

  useEffect(() => {
    const cancel = EventsOn("extract:progress", (data: ExtractProgress) => {
      const { itemId, amountExtracted, total } = data;
      const isComplete = total > 0 && amountExtracted >= total;

      if (isComplete) {
        const existingId = toastIds.current.get(itemId);
        if (existingId) {
          toast.dismiss(existingId);
          toastIds.current.delete(itemId);
        }
        return;
      }

      const description =
        isComplete ? "Extraction complete" : `Extracting... (${amountExtracted} / ${total})`;

      const toastContent = (
        <div className="flex flex-col gap-2 w-full">
          <div className="flex items-center gap-2">
            <Download className="h-4 w-4 shrink-0" />
            <span className="text-sm font-medium truncate">Downloading {itemId}</span>
          </div>
          <div className="text-xs text-muted-foreground">{description}</div>
          {total != amountExtracted && (
            <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
              <div
                className="h-full rounded-full bg-primary transition-all duration-200"
                style={{ width: `${(amountExtracted / total) * 100}%` }}
              />
            </div>
          )}
        </div>
      );

      const existingId = toastIds.current.get(itemId);
      if (existingId) {
        toast(toastContent, { id: existingId, duration: Infinity });
      } else {
        const id = toast(toastContent, { duration: Infinity });
        toastIds.current.set(itemId, id);
      }
    });

    return cancel;
  }, []);

  return null;
}
