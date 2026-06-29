import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { api } from '../lib/api';

export function LoginPage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setErrorMessage('');
    setIsSubmitting(true);

    try {
      const response = await api.post('/api/v1/auth/login', { email, password });
      const accessToken = response.data?.data?.tokens?.accessToken;
      const refreshToken = response.data?.data?.tokens?.refreshToken;

      if (!accessToken) {
        throw new Error('ACCESS_TOKEN_MISSING');
      }

      localStorage.setItem('accessToken', accessToken);
      if (refreshToken) {
        localStorage.setItem('refreshToken', refreshToken);
      }

      navigate('/community', { replace: true });
    } catch {
      setErrorMessage('Could not sign you in. Check your email and password and try again.');
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <section className="auth-shell">
      <div className="auth-pulse auth-pulse-left" aria-hidden="true" />
      <div className="auth-pulse auth-pulse-right" aria-hidden="true" />

      <div className="auth-grid">
        <aside className="auth-copy">
          <p className="auth-eyebrow">CivicOS Access</p>
          <h1 className="auth-title">Enter your civic command center.</h1>
          <p className="auth-description">
            Track community issues, sign petitions, and keep your representatives accountable from
            one place.
          </p>
        </aside>

        <form className="auth-card" onSubmit={onSubmit}>
          <h2 className="auth-card-title">Sign in</h2>
          <p className="auth-card-subtitle">Use your CivicOS credentials to continue.</p>

          <label className="auth-label" htmlFor="email">
            Email
          </label>
          <input
            id="email"
            type="email"
            className="auth-input"
            placeholder="you@example.com"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            autoComplete="email"
            required
          />

          <label className="auth-label" htmlFor="password">
            Password
          </label>
          <input
            id="password"
            type="password"
            className="auth-input"
            placeholder="At least 8 characters"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            autoComplete="current-password"
            required
            minLength={8}
          />

          {errorMessage && <p className="auth-error">{errorMessage}</p>}

          <button type="submit" className="auth-submit" disabled={isSubmitting}>
            {isSubmitting ? 'Signing in...' : 'Sign in'}
          </button>

          <p className="auth-footer">
            No account yet?{' '}
            <Link to="/register" className="auth-link">
              Create one now
            </Link>
          </p>
        </form>
      </div>
    </section>
  );
}
