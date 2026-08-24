/**
 * Browser-side video helpers for issue attachments.
 *
 * Two jobs live here, and both are deliberately client-side:
 *
 *  - DURATION. The 20-second cap is enforced by loading the file into an
 *    HTMLVideoElement and reading `duration`. The server checks size and type
 *    but NOT duration — doing so would mean ffprobe or a container parser, and
 *    server-side media processing is out of scope. So a crafted request can
 *    still store a long clip; the size cap bounds the real cost, and duration
 *    is a UX rule rather than a safety one. The gap is accepted knowingly.
 *
 *  - POSTER FRAME. Extracted by seeking to ~0.1s and drawing to a canvas, then
 *    uploaded as an ordinary image. This keeps every frame of video decoding on
 *    the device that already has the file, and means the server never needs to
 *    know what a video is beyond bytes with a MIME type.
 */

export const MAX_VIDEO_MB = 10;
export const MAX_VIDEO_SECONDS = 20;

export const ACCEPTED_VIDEO_TYPES = ['video/mp4', 'video/webm', 'video/quicktime'] as const;

/** Matches the `accept` attribute to the server's allowlist. */
export const VIDEO_ACCEPT = 'video/mp4,video/webm,video/quicktime,.mp4,.webm,.mov';

/** Poster frames are small on purpose — they load on cellular before playback. */
const POSTER_MAX_WIDTH = 640;
const POSTER_QUALITY = 0.72;
const POSTER_SEEK_SECONDS = 0.1;

/** Guards against a decode that never fires an event on a malformed file. */
const METADATA_TIMEOUT_MS = 15_000;

export class VideoReadError extends Error {}

/**
 * Loads a File into a detached <video> and hands it to `use`, cleaning up the
 * element and its object URL afterwards regardless of outcome.
 */
async function withVideoElement<T>(
  file: File,
  use: (video: HTMLVideoElement) => Promise<T>,
): Promise<T> {
  const url = URL.createObjectURL(file);
  const video = document.createElement('video');
  video.preload = 'metadata';
  video.muted = true;
  // Required for iOS Safari to decode without entering fullscreen playback.
  video.playsInline = true;
  video.src = url;

  try {
    await waitForEvent(video, 'loadedmetadata');
    return await use(video);
  } finally {
    video.removeAttribute('src');
    video.load();
    URL.revokeObjectURL(url);
  }
}

function waitForEvent(video: HTMLVideoElement, event: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const timer = window.setTimeout(() => {
      cleanup();
      reject(new VideoReadError(`Timed out waiting for "${event}"`));
    }, METADATA_TIMEOUT_MS);

    function cleanup() {
      window.clearTimeout(timer);
      video.removeEventListener(event, onDone);
      video.removeEventListener('error', onError);
    }
    function onDone() {
      cleanup();
      resolve();
    }
    function onError() {
      cleanup();
      reject(new VideoReadError('The browser could not read this video'));
    }

    video.addEventListener(event, onDone, { once: true });
    video.addEventListener('error', onError, { once: true });
  });
}

/**
 * Reads the clip's duration in seconds. Throws VideoReadError if the browser
 * cannot decode the file or reports a non-finite duration (some streamed WebM
 * files report Infinity until fully buffered).
 */
export async function readVideoDuration(file: File): Promise<number> {
  return withVideoElement(file, async (video) => {
    const { duration } = video;
    if (!Number.isFinite(duration) || duration <= 0) {
      throw new VideoReadError('The browser could not determine this video’s length');
    }
    return duration;
  });
}

/**
 * Grabs a frame near the start of the clip as a JPEG Blob, suitable for
 * uploading through the ordinary image endpoint.
 *
 * Returns null rather than throwing when the frame can't be captured — a
 * missing poster degrades to a neutral placeholder, which is not worth failing
 * an otherwise valid report over.
 */
export async function capturePosterFrame(file: File): Promise<Blob | null> {
  try {
    return await withVideoElement(file, async (video) => {
      // Seeking past the very first frame avoids the black or garbled frame
      // many encoders put at 0s.
      video.currentTime = Math.min(POSTER_SEEK_SECONDS, Math.max(video.duration - 0.01, 0));
      await waitForEvent(video, 'seeked');

      const width = video.videoWidth;
      const height = video.videoHeight;
      if (!width || !height) return null;

      const scale = Math.min(1, POSTER_MAX_WIDTH / width);
      const canvas = document.createElement('canvas');
      canvas.width = Math.round(width * scale);
      canvas.height = Math.round(height * scale);

      const ctx = canvas.getContext('2d');
      if (!ctx) return null;
      ctx.drawImage(video, 0, 0, canvas.width, canvas.height);

      return await new Promise<Blob | null>((resolve) => {
        canvas.toBlob((blob) => resolve(blob), 'image/jpeg', POSTER_QUALITY);
      });
    });
  } catch {
    return null;
  }
}

/** Wraps a poster Blob as a File so it can go through uploadImage unchanged. */
export function posterAsFile(blob: Blob): File {
  return new File([blob], 'poster.jpg', { type: 'image/jpeg' });
}

/** Human-readable byte count, e.g. "4.2 MB". Locale-formatted for the number. */
export function formatBytes(bytes: number, locale?: string): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '';
  const mb = bytes / (1024 * 1024);
  if (mb >= 1) {
    return `${mb.toLocaleString(locale, { maximumFractionDigits: 1 })} MB`;
  }
  const kb = bytes / 1024;
  return `${kb.toLocaleString(locale, { maximumFractionDigits: 0 })} KB`;
}
