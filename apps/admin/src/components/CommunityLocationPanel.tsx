import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { MapPin, ExternalLink, TriangleAlert } from 'lucide-react';
import { apiPatch } from '../lib/api';
import { looksOutsideNigeria, mapsHref, parseCoordinates } from './CoordinateField';

/**
 * Sets the point a community sits at.
 *
 * This exists because flood forecasts arrive as coordinates and everything
 * else CivicOS knows about a community is an administrative name. Until an
 * admin sets a point here, that community is skipped by the flood matcher
 * entirely — silently, and correctly, because warning the wrong town is
 * worse than warning nobody.
 *
 * Before this panel the only way to supply coordinates was a hand-rolled
 * PATCH, which is not an operational answer for the people who will
 * actually do it.
 */

export interface Located {
  latitude?: number | null;
  longitude?: number | null;
}

export function CommunityLocationPanel({
  communityId,
  community,
}: {
  communityId: string;
  community: Located;
}) {
  const queryClient = useQueryClient();
  const hasPoint =
    typeof community.latitude === 'number' && typeof community.longitude === 'number';

  const [paste, setPaste] = useState('');
  const [error, setError] = useState<string | null>(null);

  const parsed = parseCoordinates(paste);
  const showParseError = paste.trim().length > 0 && parsed === null;

  const save = useMutation({
    mutationFn: (body: Record<string, unknown>) =>
      apiPatch(`/api/v1/communities/${communityId}`, body),
    onSuccess: () => {
      setPaste('');
      setError(null);
      void queryClient.invalidateQueries({ queryKey: ['admin-community', communityId] });
      void queryClient.invalidateQueries({ queryKey: ['admin-communities'] });
    },
    onError: (err: unknown) => {
      const body = (err as { response?: { data?: { message?: string } } })?.response?.data;
      setError(body?.message ?? 'Could not save. Please try again.');
    },
  });

  return (
    <>
      <h2
        className="text-xs font-semibold text-slate-500 mono mt-6 mb-2"
        style={{ letterSpacing: '0.16em' }}
      >
        LOCATION
      </h2>

      <div className="admin-table-shell p-4">
        {hasPoint ? (
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="flex items-center gap-1.5 text-sm font-medium text-slate-800">
                <MapPin className="h-4 w-4 text-emerald-600" aria-hidden="true" />
                <span className="mono">
                  {community.latitude}, {community.longitude}
                </span>
              </p>
              <p className="mt-1 text-xs text-slate-500">
                Flood forecasts are matched against this point.
              </p>
              {/* CivicOS has no map surface, so verification happens
                  elsewhere. A wrong point is invisible here and obvious on
                  a map — this is the cheapest way to catch it. */}
              <a
                className="mt-1 inline-flex items-center gap-1 text-xs font-medium text-civic-700 hover:underline"
                href={mapsHref(community.latitude!, community.longitude!)}
                target="_blank"
                rel="noreferrer noopener"
              >
                Check this point on a map
                <ExternalLink className="h-3 w-3" aria-hidden="true" />
              </a>
              {looksOutsideNigeria(community.latitude!, community.longitude!) && (
                <p className="mt-2 flex items-start gap-1.5 text-xs text-amber-700">
                  <TriangleAlert className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                  This point is outside Nigeria. Check it before relying on flood alerts here.
                </p>
              )}
            </div>
            <button
              type="button"
              className="text-xs font-medium text-rose-600 hover:underline"
              disabled={save.isPending}
              onClick={() => {
                if (
                  window.confirm(
                    'Remove this point? The community will stop receiving flood forecasts.',
                  )
                ) {
                  save.mutate({ clearCoordinates: true });
                }
              }}
            >
              Remove point
            </button>
          </div>
        ) : (
          <p className="flex items-start gap-2 rounded-lg bg-amber-50 p-3 text-sm text-amber-900">
            <TriangleAlert className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
            <span>
              No point set. This community is <strong>excluded from flood forecasts</strong> — it
              will never show a flood warning, however severe.
            </span>
          </p>
        )}

        <form
          className="mt-4 border-t border-slate-200 pt-4"
          onSubmit={(e) => {
            e.preventDefault();
            if (!parsed) return;
            save.mutate({ latitude: parsed.lat, longitude: parsed.lng });
          }}
        >
          <label className="block text-xs font-semibold text-slate-600" htmlFor="coords">
            {hasPoint ? 'Replace with' : 'Set a point'}
          </label>
          <input
            id="coords"
            className="admin-input mt-1 w-full"
            placeholder="7.7322, 8.5391 — or paste a Google Maps link"
            value={paste}
            onChange={(e) => setPaste(e.target.value)}
          />
          <p className="mt-1 text-xs text-slate-500">
            In Google Maps, right-click the place and click the coordinates to copy them. Pasting
            the map link works too.
          </p>

          {showParseError && (
            <p className="mt-2 text-xs text-rose-600">
              Could not read a coordinate from that. Expected something like{' '}
              <span className="mono">7.7322, 8.5391</span>.
            </p>
          )}

          {parsed && (
            <div className="mt-2 rounded-lg bg-slate-50 p-3 text-xs">
              <p className="text-slate-700">
                Will save{' '}
                <span className="mono font-semibold">
                  {parsed.lat}, {parsed.lng}
                </span>
              </p>
              {/* Confirm before saving, not after. Once stored, a wrong
                  point looks exactly like a right one. */}
              <a
                className="mt-1 inline-flex items-center gap-1 font-medium text-civic-700 hover:underline"
                href={mapsHref(parsed.lat, parsed.lng)}
                target="_blank"
                rel="noreferrer noopener"
              >
                Check it on a map first
                <ExternalLink className="h-3 w-3" aria-hidden="true" />
              </a>
              {looksOutsideNigeria(parsed.lat, parsed.lng) && (
                <p className="mt-1.5 flex items-start gap-1.5 text-amber-700">
                  <TriangleAlert className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                  This is outside Nigeria. Did you swap latitude and longitude?
                </p>
              )}
            </div>
          )}

          {error && <p className="mt-2 text-xs text-rose-600">{error}</p>}

          <button
            type="submit"
            className="admin-btn admin-btn-primary mt-3"
            disabled={!parsed || save.isPending}
          >
            {save.isPending ? 'Saving…' : hasPoint ? 'Replace point' : 'Save point'}
          </button>
        </form>
      </div>
    </>
  );
}
