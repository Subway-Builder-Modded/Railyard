import { useState } from "react";
import { useConfigStore } from "@/stores/config-store";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  TrainTrack,
  FolderSearch,
  Gamepad2,
  CheckCircle,
  XCircle,
  Loader2,
  ChevronRight,
} from "lucide-react";

export function SetupScreen() {
  const [step, setStep] = useState(1);
  const [saving, setSaving] = useState(false);
  const { config, validation, openDataFolderDialog, openExecutableDialog, saveConfig } =
    useConfigStore();

  const handleDataFolder = async (autoDetect: boolean) => {
    try {
      const result = await openDataFolderDialog(autoDetect);
      if (result.source === "cancelled") return;
    } catch {
      // error is set in the store
    }
  };

  const handleExecutable = async (autoDetect: boolean) => {
    try {
      const result = await openExecutableDialog(autoDetect);
      if (result.source === "cancelled") return;
    } catch {
      // error is set in the store
    }
  };

  const handleFinish = async () => {
    setSaving(true);
    try {
      await saveConfig();
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-background">
      <Card className="w-full max-w-lg">
        <CardHeader className="text-center">
          {step === 1 ? (
            <>
              <div className="mx-auto mb-2">
                <TrainTrack className="h-10 w-10 text-primary" />
              </div>
              <CardTitle className="text-2xl">Welcome to Railyard</CardTitle>
              <CardDescription>
                Set the path to your MetroMaker data folder.
              </CardDescription>
            </>
          ) : (
            <>
              <div className="mx-auto mb-2">
                <Gamepad2 className="h-10 w-10 text-primary" />
              </div>
              <CardTitle className="text-2xl">Game Executable</CardTitle>
              <CardDescription>
                Set the path to your game executable.
              </CardDescription>
            </>
          )}

          {/* Step indicators */}
          <div className="flex items-center justify-center gap-2 pt-2">
            <span
              className={`h-2 w-6 rounded-full ${
                step === 1 ? "bg-primary" : "bg-muted"
              }`}
            />
            <span
              className={`h-2 w-6 rounded-full ${
                step === 2 ? "bg-primary" : "bg-muted"
              }`}
            />
          </div>
        </CardHeader>

        <CardContent className="space-y-6">
          {step === 1 ? (
            <>
              <div className="flex flex-col gap-3 sm:flex-row">
                <Button
                  className="flex-1"
                  onClick={() => handleDataFolder(true)}
                >
                  <FolderSearch className="mr-2 h-4 w-4" />
                  Auto-detect
                </Button>
                <Button
                  variant="outline"
                  className="flex-1"
                  onClick={() => handleDataFolder(false)}
                >
                  Browse...
                </Button>
              </div>

              {config?.metroMakerDataPath && (
                <div className="space-y-2">
                  <code className="block rounded bg-muted px-3 py-2 text-sm font-mono break-all">
                    {config.metroMakerDataPath}
                  </code>
                  {validation?.metroMakerDataPathValid ? (
                    <Badge variant="secondary" className="gap-1 text-green-600">
                      <CheckCircle className="h-3 w-3" />
                      Valid
                    </Badge>
                  ) : (
                    <Badge variant="secondary" className="gap-1 text-red-600">
                      <XCircle className="h-3 w-3" />
                      Invalid
                    </Badge>
                  )}
                </div>
              )}

              <div className="flex justify-end">
                <Button
                  onClick={() => setStep(2)}
                  disabled={!validation?.metroMakerDataPathValid}
                >
                  Next
                  <ChevronRight className="ml-2 h-4 w-4" />
                </Button>
              </div>
            </>
          ) : (
            <>
              <div className="flex flex-col gap-3 sm:flex-row">
                <Button
                  className="flex-1"
                  onClick={() => handleExecutable(true)}
                >
                  <FolderSearch className="mr-2 h-4 w-4" />
                  Auto-detect
                </Button>
                <Button
                  variant="outline"
                  className="flex-1"
                  onClick={() => handleExecutable(false)}
                >
                  Browse...
                </Button>
              </div>

              {config?.executablePath && (
                <div className="space-y-2">
                  <code className="block rounded bg-muted px-3 py-2 text-sm font-mono break-all">
                    {config.executablePath}
                  </code>
                  {validation?.executablePathValid ? (
                    <Badge variant="secondary" className="gap-1 text-green-600">
                      <CheckCircle className="h-3 w-3" />
                      Valid
                    </Badge>
                  ) : (
                    <Badge variant="secondary" className="gap-1 text-red-600">
                      <XCircle className="h-3 w-3" />
                      Invalid
                    </Badge>
                  )}
                </div>
              )}

              <div className="flex justify-between">
                <Button variant="outline" onClick={() => setStep(1)}>
                  Back
                </Button>
                <Button
                  onClick={handleFinish}
                  disabled={!validation?.executablePathValid || saving}
                >
                  {saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  Finish Setup
                </Button>
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
