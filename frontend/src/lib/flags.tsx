import { type ComponentType, type SVGProps } from "react";
import * as FlagIcons from "country-flag-icons/react/3x2";

type FlagComponent = ComponentType<SVGProps<SVGSVGElement>>;

const FLAGS_BY_COUNTRY = FlagIcons as unknown as Record<string, FlagComponent>;

export function getCountryFlagIcon(countryCode: string | undefined): FlagComponent | null {
  const normalized = (countryCode ?? "").trim().toUpperCase();
  if (!/^[A-Z]{2}$/.test(normalized)) {
    return null;
  }

  return FLAGS_BY_COUNTRY[normalized] ?? null;
}

