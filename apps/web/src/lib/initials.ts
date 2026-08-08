/**
 * Two-letter initials from a free-text display name.
 *
 * Used by the dashboard avatar (and anything else that needs to
 * summarise a user by their initials). Rules:
 *
 *   - Whitespace splits — "Gino Osahon" → ["Gino", "Osahon"]
 *   - First letter of each part, uppercased, up to two
 *   - Single-word names use the first letter only ("ada" → "A")
 *   - Empty / missing name returns "?" so the avatar circle never
 *     reads blank
 *
 * Returns a string of length 1 or 2 — never more, never zero. Doesn't
 * try to handle middle initials or accented characters specially; this
 * is meant for a tiny circle, not a name parser.
 */
export function meInitials(name: string | null | undefined): string {
  if (!name) return '?';
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  if (parts.length === 1) return parts[0].slice(0, 1).toUpperCase();
  return (parts[0].slice(0, 1) + parts[parts.length - 1].slice(0, 1)).toUpperCase();
}
