import { useTranslation } from 'react-i18next';
import { Film } from 'lucide-react';
import { uploadUrl } from '../lib/api';
import { formatBytes } from '../lib/video';

/**
 * Playback for an issue's video attachment.
 *
 * The constraints here are accessibility requirements, not styling choices —
 * a large share of CivicOS readers are on metered mobile data:
 *
 *  - `preload="none"` so not one byte of video is fetched until the reader
 *    presses play. (Only the small poster JPEG loads with the page.)
 *  - no autoplay, ever, and native `controls` so the platform's own
 *    accessibility affordances — keyboard, screen reader, captions menu —
 *    all work without reimplementation.
 *  - the poster frame and the file size are both visible BEFORE playback, so
 *    the decision to spend the data is an informed one.
 */
export function IssueVideo({
  filename,
  posterFilename,
  sizeBytes,
}: {
  filename: string;
  posterFilename?: string;
  sizeBytes?: number;
}) {
  const { t, i18n } = useTranslation();

  if (!filename) return null;

  const poster = posterFilename ? uploadUrl(posterFilename) : undefined;
  const size = sizeBytes && sizeBytes > 0 ? formatBytes(sizeBytes, i18n.language) : '';

  return (
    <figure className="mt-4">
      <div className="relative overflow-hidden rounded-xl bg-slate-900 ring-1 ring-slate-200 dark:ring-slate-700">
        {/* Sits behind the video element so a clip whose poster failed to
            generate still shows something intentional rather than a black box. */}
        {!poster && (
          <div
            aria-hidden="true"
            className="pointer-events-none absolute inset-0 flex items-center justify-center text-slate-500"
          >
            <Film className="h-10 w-10" />
          </div>
        )}
        <video
          className="relative max-h-[70vh] w-full"
          src={uploadUrl(filename)}
          poster={poster}
          controls
          preload="none"
          playsInline
        >
          {t('issueDetail.videoUnsupported')}
        </video>
      </div>
      <figcaption className="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-slate-600 dark:text-slate-400">
        <span className="font-medium">{t('issueDetail.videoLabel')}</span>
        {size && (
          <>
            <span aria-hidden="true">·</span>
            <span>{size}</span>
          </>
        )}
        <span aria-hidden="true">·</span>
        <span>{t('issueDetail.videoDataHint')}</span>
      </figcaption>
    </figure>
  );
}
