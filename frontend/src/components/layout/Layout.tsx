import React from "react";
import { GetCurrentVersion } from "../../../wailsjs/go/main/App";
import { Navbar } from "./Navbar";

export function Layout({ children }: { children: React.ReactNode }) {
  const [version, setVersion] = React.useState<string>("");
  React.useMemo(() => {
    GetCurrentVersion().then((s) => {
      setVersion([...s].filter((c) => c !== "\u0000").join(""));
    });
  }, []);
  return (
    <div className="min-h-screen bg-background flex flex-col">
      <Navbar />
      <main className="max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-6 flex-1">
        {children}
      </main>
      <footer className="text-center text-sm text-muted-foreground py-4">
        <p>Railyard {version} | &copy; Subway Builder Modded 2026.</p>
      </footer>
    </div>
  );
}
