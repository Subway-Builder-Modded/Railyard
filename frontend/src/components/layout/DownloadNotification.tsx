import { useEffect, useRef } from "react";
import { EventsOn } from "../../../wailsjs/runtime/runtime";
import { toast } from "sonner";
import { Download, CheckCircle } from "lucide-react";
import { useDownloadQueueStore } from "@/stores/download-queue-store";

interface DownloadProgress {
  itemId: string;
  received: number;
  total: number;
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function DownloadNotification() {
  const toastIds = useRef<Map<string, string | number>>(new Map());

  useEffect(() => {
    const cancel = EventsOn("download:progress", (data: DownloadProgress) => {
      const { total: queueTotal } = useDownloadQueueStore.getState();
      if (queueTotal <= 0) {
        return;
      }

      const { itemId, received, total } = data;
      const percent = total > 0 ? Math.round((received / total) * 100) : -1;
      const isComplete = total > 0 && received >= total;

      if (isComplete) {
        const existingId = toastIds.current.get(itemId);
        if (existingId) {
          // Show brief "Downloaded" state before dismissing
          const { completed, total: queueTotalForLabel } = useDownloadQueueStore.getState();
          const queueLabel = queueTotalForLabel > 1 ? `${completed + 1}/${queueTotalForLabel}` : null;

          toast(
            <div className="flex flex-col gap-1.5 w-full">
              <div className="flex items-center justify-between gap-2">
                <div className="flex items-center gap-2 min-w-0">
                  <CheckCircle className="h-4 w-4 shrink-0 text-primary" />
                  <span className="text-sm font-medium truncate">Downloaded {itemId}</span>
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

      const description =
        percent >= 0
          ? `${formatBytes(received)} / ${formatBytes(total)} (${percent}%)`
          : `${formatBytes(received)} downloaded`;

      const toastContent = (
        <div className="flex flex-col gap-2 w-full">
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2 min-w-0">
              <Download className="h-4 w-4 shrink-0" />
              <span className="text-sm font-medium truncate">Downloading {itemId}</span>
            </div>
            {queueLabel && (
              <span className="text-xs font-medium text-muted-foreground shrink-0 tabular-nums">
                {queueLabel}
              </span>
            )}
          </div>
          <div className="text-xs text-muted-foreground">{description}</div>
          {percent >= 0 && (
            <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
              <div
                className="h-full rounded-full bg-primary transition-all duration-200"
                style={{ width: `${percent}%` }}
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
