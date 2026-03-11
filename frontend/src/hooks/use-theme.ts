import { useEffect } from "react";
import { useProfileStore } from "@/stores/profile-store";

export function useTheme() {
  const theme = useProfileStore((s) => s.profile?.uiPreferences?.theme ?? "system");

  useEffect(() => {
    const root = document.documentElement;

    if (!root.classList.contains("theme-ready")) {
      requestAnimationFrame(() => root.classList.add("theme-ready"));
    }

    if (theme === "system") {
      const mql = window.matchMedia("(prefers-color-scheme: dark)");
      root.classList.toggle("dark", mql.matches);

      const handler = (e: MediaQueryListEvent) => {
        root.classList.toggle("dark", e.matches);
      };
      mql.addEventListener("change", handler);
      return () => mql.removeEventListener("change", handler);
    }

    root.classList.toggle("dark", theme === "dark");
  }, [theme]);
}
