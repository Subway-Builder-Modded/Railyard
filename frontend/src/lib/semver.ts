import semver from "semver";

export function isCompatible(gameVersion: string, requiredRange: string): boolean | null {
  if (!gameVersion || !requiredRange) return null;
  const coerced = semver.coerce(gameVersion);
  if (!coerced) return null;
  try {
    return semver.satisfies(coerced, requiredRange);
  } catch {
    return null;
  }
}
