import { useEffect, useState } from 'react';
import { ExternalLink, MapPin, TriangleAlert, Sparkles } from 'lucide-react';

/**
 * Where an admin supplies the point a community sits at.
 *
 * Used on both the create form and the detail page, so the parsing and the
 * warnings behave identically in both. Two copies would drift, and the one
 * that drifts is the one that accepts a bad point.
 *
 * The whole component is built around one idea: a coordinate that is
 * plausible but wrong is invisible. It looks exactly like a correct one,
 * and the only symptom is that a town silently never receives a flood
 * warning — or receives one meant for somewhere else. So nothing here is
 * saved without the admin being offered a way to look at it on a map
 * first.
 */

export interface Suggestion {
  latitude: number;
  longitude: number;
  formattedAddress: string;
  locationType: string;
  partialMatch: boolean;
}

/**
 * Pulls a lat/lng out of whatever was pasted.
 *
 * Nobody types two decimals into two boxes — they copy from Google Maps,
 * which gives either "7.7322, 8.5391" or a URL with the point inside it.
 * Accepting both removes the transcription step, and transcription is
 * where a digit gets dropped and a town moves 100km.
 *
 * Returns null rather than guessing. A half-parsed coordinate is worse
 * than none.
 */
export function parseCoordinates(input: string): { lat: number; lng: number } | null {
  const text = input.trim();
  if (!text) return null;

  // Maps URLs put the map centre after @, and a searched place in a
  // q=/query=/ll= parameter. The centre is what a dropped pin sits on.
  const at = text.match(/@(-?\d+(?:\.\d+)?),(-?\d+(?:\.\d+)?)/);
  const q = text.match(/[?&](?:q|query|ll)=(-?\d+(?:\.\d+)?),\s*(-?\d+(?:\.\d+)?)/);
  const bare = text.match(/^(-?\d+(?:\.\d+)?)[,\s]+(-?\d+(?:\.\d+)?)$/);

  const m = at ?? q ?? bare;
  if (!m) return null;

  const lat = Number.parseFloat(m[1]);
  const lng = Number.parseFloat(m[2]);
  if (!Number.isFinite(lat) || !Number.isFinite(lng)) return null;
  if (lat < -90 || lat > 90 || lng < -180 || lng > 180) return null;
  // 0,0 is the Gulf of Guinea and is what uninitialised floats look like.
  // The server rejects it too; catching it here saves a round trip.
  if (lat === 0 && lng === 0) return null;

  return { lat, lng };
}

/** Rough bounds of Nigeria. Warns, never blocks. */
export function looksOutsideNigeria(lat: number, lng: number): boolean {
  return lat < 4 || lat > 14 || lng < 2.5 || lng > 15;
}

export function mapsHref(lat: number, lng: number): string {
  return `https://www.google.com/maps/search/?api=1&query=${lat},${lng}`;
}

export function CoordinateField({
  value,
  onChange,
  suggestion,
  status,
  onRetry,
  canRetry,
}: {
  value: { lat: number; lng: number } | null;
  onChange: (next: { lat: number; lng: number } | null) => void;
  suggestion?: Suggestion | null;
  status?: 'idle' | 'loading' | 'unavailable' | 'none';
  onRetry?: () => void;
  canRetry?: boolean;
}) {
  const [text, setText] = useState('');

  // A suggestion arriving fills the box, so the admin sees the value they
  // are about to save rather than an empty field beside a claim.
  useEffect(() => {
    if (!suggestion) return;
    setText(`${suggestion.latitude}, ${suggestion.longitude}`);
    onChange({ lat: suggestion.latitude, lng: suggestion.longitude });
  }, [suggestion, onChange]);

  const parsed = parseCoordinates(text);
  const showParseError = text.trim().length > 0 && parsed === null;
  const point = parsed ?? value;

  // Parsed on every keystroke and pushed straight up, rather than through
  // an effect: the parent needs the value at submit, and an effect keyed
  // on `text` would deliver it a render late.
  function update(next: string) {
    setText(next);
    onChange(parseCoordinates(next));
  }

  return (
    <div className="flex flex-col gap-1">
      <label className="text-sm font-medium text-slate-700" htmlFor="c-coords">
        Coordinates <span className="text-xs text-slate-500">(optional)</span>
      </label>

      <input
        id="c-coords"
        className="admin-table-search"
        placeholder="7.7322, 8.5391 — or paste a Google Maps link"
        value={text}
        onChange={(e) => update(e.target.value)}
      />

      <p className="text-xs text-slate-500">
        Without a point this community is excluded from flood forecasts. You can add it later.
      </p>

      {status === 'loading' && (
        <p className="flex items-center gap-1.5 text-xs text-slate-500">
          <Sparkles className="h-3.5 w-3.5 animate-pulse" aria-hidden="true" />
          Looking up this LGA…
        </p>
      )}

      {status === 'none' && (
        <p className="text-xs text-slate-500">
          Could not find that LGA automatically — enter the point manually.
          {canRetry && onRetry && (
            <button type="button" className="ml-1 font-medium underline" onClick={onRetry}>
              Try again
            </button>
          )}
        </p>
      )}

      {status === 'unavailable' && (
        <p className="text-xs text-slate-500">
          Automatic lookup is not configured on this deployment — enter the point manually.
        </p>
      )}

      {suggestion && parsed && (
        <p className="text-xs text-slate-600">
          Suggested from <span className="font-medium">{suggestion.formattedAddress}</span>
          {suggestion.locationType === 'APPROXIMATE' && ' (approximate)'}
          {suggestion.partialMatch && ' — partial match'}
        </p>
      )}

      {showParseError && (
        <p className="text-xs text-rose-600">
          Could not read a coordinate from that. Expected something like{' '}
          <span className="mono">7.7322, 8.5391</span>.
        </p>
      )}

      {point && (
        <div className="rounded-lg bg-slate-50 p-3 text-xs">
          <p className="flex items-center gap-1.5 text-slate-700">
            <MapPin className="h-3.5 w-3.5" aria-hidden="true" />
            <span className="mono font-semibold">
              {point.lat}, {point.lng}
            </span>
          </p>
          {/* Confirm before saving, not after. A suggested LGA centre is a
              starting point — it can easily sit in farmland miles from the
              town, and that is invisible from here. */}
          <a
            className="mt-1 inline-flex items-center gap-1 font-medium text-civic-700 hover:underline"
            href={mapsHref(point.lat, point.lng)}
            target="_blank"
            rel="noreferrer noopener"
          >
            Check this point on a map before saving
            <ExternalLink className="h-3 w-3" aria-hidden="true" />
          </a>
          {looksOutsideNigeria(point.lat, point.lng) && (
            <p className="mt-1.5 flex items-start gap-1.5 text-amber-700">
              <TriangleAlert className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
              This is outside Nigeria. Did you swap latitude and longitude?
            </p>
          )}
        </div>
      )}
    </div>
  );
}
