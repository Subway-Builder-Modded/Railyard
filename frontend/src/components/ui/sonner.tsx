import { Toaster as Sonner } from 'sonner';

import { useProfileStore } from '@/stores/profile-store';

type ToasterProps = React.ComponentProps<typeof Sonner>;

const NonDraggableSonner = Sonner as React.ComponentType<
  ToasterProps & { draggable?: boolean }
>;

function Toaster({ ...props }: ToasterProps) {
  const selectedTheme = useProfileStore((s) => s.theme)();

  const theme: ToasterProps['theme'] =
    selectedTheme === 'system'
      ? 'system'
      : selectedTheme.startsWith('dark')
        ? 'dark'
        : 'light';

  return (
    <>
      <style>{`
        [data-sonner-toast][data-swiping="true"] {
          transform: translateY(var(--y)) scale(var(--scale)) !important;
          opacity: 1 !important;
          transition: none !important;
        }
      `}</style>
      <NonDraggableSonner
        theme={theme}
        className="toaster group !z-[2147483647]"
        expand
        gap={8}
        visibleToasts={5}
        draggable={false}
        style={{ zIndex: 2147483647 }}
        toastOptions={{
          style: { zIndex: 2147483647 },
          classNames: {
            toast:
              'group toast group-[.toaster]:bg-background group-[.toaster]:text-foreground group-[.toaster]:border-border group-[.toaster]:shadow-lg',
            description: 'group-[.toast]:text-muted-foreground',
            actionButton:
              'group-[.toast]:bg-primary group-[.toast]:text-primary-foreground',
            cancelButton:
              'group-[.toast]:bg-muted group-[.toast]:text-muted-foreground',
          },
        }}
        {...props}
      />
    </>
  );
}

export { Toaster };
