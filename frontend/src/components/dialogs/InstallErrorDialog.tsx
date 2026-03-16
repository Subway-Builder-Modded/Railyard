import { Check, CircleX, Copy } from 'lucide-react';
import { useState } from 'react';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';

interface InstallErrorDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  itemName: string;
  version: string;
  error: string;
}

export function InstallErrorDialog({
  open,
  onOpenChange,
  itemName,
  version,
  error,
}: InstallErrorDialogProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(error);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <CircleX className="h-5 w-5 text-destructive" />
            Installation Failed
          </DialogTitle>
          <DialogDescription>
            Failed to install{' '}
            <span className="font-semibold text-foreground">{itemName}</span>{' '}
            {version}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-0">
          <div className="flex items-center justify-between rounded-t-md border border-b-0 border-border bg-muted px-3 py-1.5">
            <span className="text-xs font-medium text-muted-foreground">
              Error Details
            </span>
            <Button
              variant="ghost"
              size="sm"
              className="h-6 gap-1.5 px-2 text-xs text-muted-foreground hover:text-foreground"
              onClick={handleCopy}
            >
              {copied ? (
                <Check className="h-3 w-3" />
              ) : (
                <Copy className="h-3 w-3" />
              )}
              {copied ? 'Copied' : 'Copy'}
            </Button>
          </div>
          <pre className="rounded-b-md border border-t-0 border-border bg-muted/50 p-4 text-xs font-mono whitespace-pre-wrap break-all max-h-60 overflow-y-auto">
            {error}
          </pre>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
