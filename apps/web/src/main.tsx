import { StrictMode, Suspense, lazy } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import App from './App';
import './i18n';
import './index.css';

// Devtools are lazy-loaded and only mounted in dev builds. In prod the
// `import.meta.env.DEV` short-circuit keeps the Suspense/Lazy pair
// from ever fetching the chunk, and Vite splits it into a separate
// module so it isn't in the initial JS payload either.
const ReactQueryDevtools = lazy(() =>
  import('@tanstack/react-query-devtools').then((m) => ({ default: m.ReactQueryDevtools })),
);

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60, // 1 minute
      // retry: false is load-bearing. Do not "improve" this back to a
      // retry count without re-testing an offline page load.
      //
      // With ANY retry enabled (the previous `retry: 1`, or a predicate),
      // a failed query does not end up in the error state. It ends up
      // stranded at status 'pending' / fetchStatus 'paused' with
      // fetchFailureCount 1 — and it stays there. Forever. Observed
      // directly against a downed service with @tanstack/react-query 5.45:
      // the first attempt fails, a retry is scheduled, the retryer pauses,
      // and nothing resumes it — not a reconnect, not refetch(), not the
      // Retry button.
      //
      // The user-visible result was the bug this whole change exists to fix:
      // isLoading false AND isError false, so pages rendered neither a
      // spinner nor an error. List pages fell through to "No projects here
      // yet"; detail pages fell through to their generic load error. Someone
      // whose connection dropped was told their community was empty.
      //
      // Failing fast is strictly better for a civic tool used on unreliable
      // mobile data: one honest error, a working Retry button, and
      // refetchOnReconnect below to heal automatically. A silent retry that
      // can strand the query is not worth the one saved request.
      retry: false,
      // Reconnecting should heal the page by itself rather than leaving a
      // wall of stale error panels the user has to reload past.
      refetchOnReconnect: true,
      // Second half of the same defence. TanStack's default networkMode is
      // 'online', which pauses rather than fails whenever it believes there
      // is no connection. 'always' sends the request regardless so it fails
      // honestly and lands in the error path, where classifyError() can name
      // the cause. One code path for every failure, no silent third state.
      networkMode: 'always',
    },
    mutations: {
      // Same reasoning: a paused mutation looks to the user like a button
      // that did nothing at all.
      networkMode: 'always',
    },
  },
});

const root = document.getElementById('root');
if (!root) throw new Error('Root element not found');

createRoot(root).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
      {import.meta.env.DEV && (
        <Suspense fallback={null}>
          <ReactQueryDevtools initialIsOpen={false} />
        </Suspense>
      )}
    </QueryClientProvider>
  </StrictMode>,
);
