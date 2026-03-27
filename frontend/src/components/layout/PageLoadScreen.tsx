import { Loader2 } from 'lucide-react';

interface PageLoadScreenProps {
  title: string;
  description: string;
}

export function PageLoadScreen({ title, description }: PageLoadScreenProps) {
  return (
    <div className="fixed inset-0 z-[120] flex min-h-screen w-full select-none items-center justify-center bg-background text-foreground">
      <div className="flex flex-col items-center gap-3 px-6 text-center">
        <Loader2 className="h-6 w-6 animate-spin text-primary" />
        <p className="text-base font-medium">{title}</p>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
    </div>
  );
}
