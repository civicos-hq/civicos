/**
 * Error classification for the citizen app.
 *
 * The logic itself is shared with the admin console — both hit the same
 * gateway and fail the same ways — so it lives in @civicos/types. Only the
 * COPY differs between the apps, and that lives in each app's own layer
 * (here: hooks/useErrorMessage.ts, backed by the `errors.*` i18n keys).
 *
 * Re-exported through this module so the ~15 call sites that already import
 * from '../lib/apiError' keep working, and so there is one obvious place to
 * add web-only error handling later.
 */
export {
  classifyError,
  isRetryableError,
  queryFailure,
  isKnownErrorCode,
  KNOWN_ERROR_CODES,
  QUERY_PAUSED,
} from '@civicos/types';

export type { ApiErrorKind, ClassifiedError, KnownErrorCode } from '@civicos/types';
