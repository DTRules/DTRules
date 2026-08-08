import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * Human-readable project name derived from its path. There is no project-name
 * field in the EDD, so use the last path segment that isn't a generic
 * directory name (xml, rules, dtrules, ...).
 */
export function deriveProjectName(path: string | null): string {
  if (!path) return ""
  const generic = new Set(["xml", "rules", "dtrules", "pkg", "src", "data", "project"])
  const segments = path.replace(/\/+$/, "").split("/").filter(Boolean)
  for (let i = segments.length - 1; i >= 0; i--) {
    if (!generic.has(segments[i].toLowerCase())) return segments[i]
  }
  return segments[segments.length - 1] || ""
}
