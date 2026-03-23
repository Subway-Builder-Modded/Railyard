import { Check, Circle, Loader2 } from 'lucide-react';

import { cn } from '@/lib/utils';

export interface LoadingState {
  text: string;
}

interface MultiStepLoaderProps {
  loadingStates: LoadingState[];
  currentStep: number;
}

export function MultiStepLoader({
  loadingStates,
  currentStep,
}: MultiStepLoaderProps) {
  return (
    <div className="min-h-screen w-full flex flex-col items-center justify-center bg-background text-foreground select-none">
      <div className="flex flex-col items-center gap-8 px-6">
        <div className="flex flex-col items-center gap-1.5">
          <h1 className="text-xl font-semibold tracking-tight text-foreground">
            Railyard
          </h1>
          <p className="text-xs text-muted-foreground">
            Getting things ready&hellip;
          </p>
        </div>

        <div className="flex flex-col gap-1 min-w-[300px] rounded-xl border border-border/60 bg-card/60 p-2 backdrop-blur-sm">
          {loadingStates.map((state, index) => {
            const isComplete = index < currentStep;
            const isActive = index === currentStep;

            return (
              <div
                key={state.text}
                className={cn(
                  'flex items-center gap-3 rounded-lg px-3 py-2.5 transition-all duration-500 ease-out',
                  isActive && 'bg-muted/60',
                  isComplete && 'opacity-75',
                  !isActive && !isComplete && 'opacity-40',
                )}
              >
                <div className="flex h-5 w-5 shrink-0 items-center justify-center">
                  {isComplete ? (
                    <div className="flex h-[18px] w-[18px] items-center justify-center rounded-full bg-chart-2/15 transition-all duration-300">
                      <Check
                        className="h-3 w-3 text-chart-2"
                        strokeWidth={2.5}
                      />
                    </div>
                  ) : isActive ? (
                    <Loader2 className="h-4 w-4 animate-spin text-foreground" />
                  ) : (
                    <Circle className="h-2 w-2 fill-muted-foreground/30 text-muted-foreground/30" />
                  )}
                </div>

                <span
                  className={cn(
                    'text-sm leading-none transition-colors duration-300',
                    isActive && 'font-medium text-foreground',
                    isComplete && 'text-muted-foreground',
                    !isActive && !isComplete && 'text-muted-foreground/70',
                  )}
                >
                  {state.text}
                </span>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
