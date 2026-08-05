import { Component, type ReactNode } from 'react';

/**
 * Catches a render error and shows something useful instead of nothing.
 *
 * Without this, one uncaught exception anywhere in the tree unmounts the whole
 * console and leaves a white page — no message, no route, nothing to report.
 * That is exactly what a null array from an API did to the funding analytics
 * page in production: the operator could see the page render and then vanish,
 * with no way to tell whether the service was down, their session had expired,
 * or the build was broken.
 *
 * Scoped to the routed content rather than the whole app, so the sidebar and
 * sign-out survive: an admin who hits a broken page can still navigate away
 * from it.
 *
 * `key`ed on the pathname by the caller, so navigating to another page clears
 * the error — otherwise the boundary latches and every subsequent route looks
 * broken too.
 */
interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: { componentStack?: string | null }) {
    // Operators report what they can see. Put the real error somewhere they
    // can copy it from, rather than only in a React dev overlay that a
    // production build does not show.
    console.error('[admin] render error:', error, info?.componentStack);
  }

  render() {
    const { error } = this.state;
    if (!error) return this.props.children;

    return (
      <section className="admin-table-shell">
        <div className="admin-table-toolbar">
          <strong className="text-sm">This page failed to render</strong>
        </div>
        <div className="p-4 text-sm">
          <p className="text-slate-600">
            Something in this page threw an error. The rest of the console still works — use the
            sidebar to go elsewhere.
          </p>
          <p className="mt-3 text-slate-600">
            If it keeps happening, send this to the team along with what you clicked:
          </p>
          <pre className="admin-error-detail">{error.message || String(error)}</pre>
          <button type="button" className="admin-btn" onClick={() => window.location.reload()}>
            Reload the page
          </button>
        </div>
      </section>
    );
  }
}
