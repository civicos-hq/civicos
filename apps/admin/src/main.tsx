import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { App } from './App';
import './index.css';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Admin data changes rarely and staleness is fine — the moderation
      // queue counts are a hint, not a live feed.
      staleTime: 30_000,
      // retry: false is load-bearing. Do not "improve" this back to a retry
      // count without re-testing against a downed service.
      //
      // With ANY retry enabled (this was `retry: 1`), a failed query never
      // reaches the error state. It strands at status 'pending' /
      // fetchStatus 'paused' and stays there — nothing resumes it, not a
      // reconnect, not refetch(), not a Retry button. isLoading and isError
      // are BOTH false, so pages render neither a spinner nor an error and
      // fall through to whatever comes last, usually an empty table. An
      // operator during an outage was shown an empty moderation queue
      // instead of "a backend service is down" — the worst possible failure
      // mode for a console whose whole job is telling you what is wrong.
      //
      // Same finding and same fix as apps/web/src/main.tsx.
      retry: false,
      // Pausing is the other half of the same trap: 'always' makes requests
      // fire regardless of what the client believes about connectivity, so
      // they fail honestly and land in the error path.
      networkMode: 'always',
      refetchOnReconnect: true,
    },
    mutations: {
      networkMode: 'always',
    },
  },
});

const root = document.getElementById('root');
if (!root) throw new Error('missing #root');

createRoot(root).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
);
