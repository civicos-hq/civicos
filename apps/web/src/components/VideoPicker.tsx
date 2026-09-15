import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Film, X } from 'lucide-react';
import {
  ACCEPTED_VIDEO_TYPES,
  MAX_VIDEO_MB,
  MAX_VIDEO_SECONDS,
  VIDEO_ACCEPT,
  formatBytes,
  readVideoDuration,
} from '../lib/video';

/**
 * Single-video picker for the report-issue modal.
 *
 * Validation runs here, before the file leaves the device, because that is the
 * only place duration can be read cheaply. Size and type are re-checked on the
 * server; duration is not (see lib/video.ts for why that gap is deliberate).
 *
 * Attaching a video is entirely optional — a text-only report is a complete
 * report, and nothing in this component nags otherwise.
 */
export function VideoPicker({
  file,
  onChange,
  disabled,
}: {
  file: File | null;
  onChange: (file: File | null) => void;
  disabled?: boolean;
}) {
  const { t, i18n } = useTranslation();
  const [error, setError] = useState('');
  const [checking, setChecking] = useState(false);
  const [poster, setPoster] = useState<string | null>(null);

  // A cheap local preview of the chosen clip. The uploaded poster is generated
  // separately at submit time; this one only has to survive the modal.
  useEffect(() => {
    if (!file) {
      setPoster(null);
      return;
    }
    const url = URL.createObjectURL(file);
    setPoster(url);
    return () => URL.revokeObjectURL(url);
  }, [file]);

  async function handlePick(e: React.ChangeEvent<HTMLInputElement>) {
    const picked = e.target.files?.[0] ?? null;
    e.target.value = '';
    if (!picked) return;

    setError('');

    if (picked.size > MAX_VIDEO_MB * 1024 * 1024) {
      setError(t('issuesPage.modal.videoTooBig', { max: MAX_VIDEO_MB }));
      return;
    }
    if (!(ACCEPTED_VIDEO_TYPES as readonly string[]).includes(picked.type)) {
      setError(t('issuesPage.modal.videoWrongType'));
      return;
    }

    setChecking(true);
    try {
      const duration = await readVideoDuration(picked);
      if (duration > MAX_VIDEO_SECONDS) {
        setError(
          t('issuesPage.modal.videoTooLong', {
            max: MAX_VIDEO_SECONDS,
            seconds: Math.round(duration),
          }),
        );
        return;
      }
      onChange(picked);
    } catch {
      setError(t('issuesPage.modal.videoUnreadable'));
    } finally {
      setChecking(false);
    }
  }

  return (
    <div className="flex flex-col gap-2">
      <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
        {t('issuesPage.modal.video')}
      </label>

      {file ? (
        <div className="relative flex items-center gap-3 rounded-lg border border-gray-200 dark:border-gray-700 p-2">
          {poster ? (
            <video
              src={poster}
              className="h-16 w-24 shrink-0 rounded-md bg-slate-900 object-cover"
              muted
              playsInline
              preload="metadata"
            />
          ) : (
            <div className="flex h-16 w-24 shrink-0 items-center justify-center rounded-md bg-slate-900 text-slate-500">
              <Film className="h-6 w-6" />
            </div>
          )}
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm text-gray-900 dark:text-gray-100">{file.name}</p>
            <p className="text-xs text-gray-500 dark:text-gray-400">
              {formatBytes(file.size, i18n.language)}
            </p>
          </div>
          <button
            type="button"
            onClick={() => {
              onChange(null);
              setError('');
            }}
            disabled={disabled}
            className="rounded-full p-1 text-gray-500 transition hover:bg-gray-100 hover:text-gray-900 dark:hover:bg-gray-700 dark:hover:text-gray-100"
            aria-label={t('issuesPage.modal.removeVideo')}
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      ) : (
        <label
          className={`flex cursor-pointer items-center justify-center rounded-lg border border-dashed border-gray-300 dark:border-gray-600 px-3 py-3 text-sm text-gray-600 dark:text-gray-400 transition hover:border-civic-400 hover:bg-civic-50 ${
            disabled || checking ? 'pointer-events-none opacity-50' : ''
          }`}
        >
          <input
            type="file"
            accept={VIDEO_ACCEPT}
            className="hidden"
            onChange={handlePick}
            disabled={disabled || checking}
          />
          {checking
            ? t('issuesPage.modal.videoChecking')
            : t('issuesPage.modal.addVideo', { max: MAX_VIDEO_MB, seconds: MAX_VIDEO_SECONDS })}
        </label>
      )}

      {error && <p className="text-xs text-rose-600 dark:text-rose-400">{error}</p>}
    </div>
  );
}
