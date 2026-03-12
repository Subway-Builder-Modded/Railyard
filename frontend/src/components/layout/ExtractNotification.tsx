import { useEffect, useRef } from "react";
import { EventsOn } from "../../../wailsjs/runtime/runtime";
import { toast } from "sonner";
import { Package, CheckCircle } from "lucide-react";
import { useDownloadQueueStore } from "@/stores/download-queue-store";

interface ExtractProgress {
    itemId: string,
    amountExtracted: number,
    total: number,
}

export function ExtractNotification() {
  const toastIds = useRef<Map<string, string | number>>(new Map());

  useEffect(() => {
    const cancel = EventsOn("extract:progress", (data: ExtractProgress) => {
      const { total: queueTotal } = useDownloadQueueStore.getState();
      if (queueTotal <= 0) {
        return;
      }

      const { itemId, amountExtracted, total } = data;
      const isComplete = total > 0 && amountExtracted >= total;

      if (isComplete) {
        const existingId = toastIds.current.get(itemId);
        if (existingId) {
          const { completed, total: queueTotalForLabel } = useDownloadQueueStore.getState();
          const queueLabel = queueTotalForLabel > 1 ? `${completed + 1}/${queueTotalForLabel}` : null;

          toast(
            <div className="flex flex-col gap-1.5 w-full">
              <div className="flex items-center justify-between gap-2">
                <div className="flex items-center gap-2 min-w-0">
                  <CheckCircle className="h-4 w-4 shrink-0 text-primary" />
                  <span className="text-sm font-medium truncate">Extracted {itemId}</span>
                </div>
                {queueLabel && (
                  <span className="text-xs font-medium text-muted-foreground shrink-0 tabular-nums">
                    {queueLabel}
                  </span>
                )}
              </div>
            </div>,
            { id: existingId, duration: 2000 },
          );
          toastIds.current.delete(itemId);
        }
        return;
      }

      const { completed, total: queueTotalForLabel } = useDownloadQueueStore.getState();
      const queueLabel = queueTotalForLabel > 1 ? `${completed + 1}/${queueTotalForLabel}` : null;

      const description = `Extracting\u2026 (${amountExtracted} / ${total})`;

      const toastContent = (
        <div className="flex flex-col gap-2 w-full">
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2 min-w-0">
              <Package className="h-4 w-4 shrink-0" />
              <span className="text-sm font-medium truncate">Extracting {itemId}</span>
            </div>
            {queueLabel && (
              <span className="text-xs font-medium text-muted-foreground shrink-0 tabular-nums">
                {queueLabel}
              </span>
            )}
          </div>
          <div className="text-xs text-muted-foreground">{description}</div>
          {total !== amountExtracted && (
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
