import { useState, type FormEvent } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { api, setSession } from '../lib/api';

export function LoginPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      // Same /auth/login endpoint the citizen app uses. We just enforce
      // an extra role gate on the response before storing the session.
      const res = await api.post<{
        success: boolean;
        data: {
          accessToken: string;
          user: { id: string; email: string; name: string; role: string; emailVerified: boolean };
        };
      }>('/api/v1/auth/login', { email, password });

      const { accessToken, user } = res.data.data;

      if (user.role !== 'PLATFORM_ADMIN') {
        setError(
          `This account (role ${user.role}) is not authorized for the admin console. Contact a platform administrator to be elevated.`,
        );
        return;
      }

      setSession({ accessToken, user });
      const redirect = searchParams.get('redirect') ?? '/';
      navigate(redirect, { replace: true });
    } catch (err) {
      const msg =
        (err as { response?: { data?: { message?: string } } }).response?.data?.message ??
        'Login failed. Check your credentials and try again.';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="admin-login">
      <div className="admin-login-card">
        <p className="admin-login-eyebrow">CivicOS · Admin Console</p>
        <h1 className="admin-login-title">Sign in</h1>
        <p className="admin-login-sub">
          Only accounts with the <span className="mono">PLATFORM_ADMIN</span> role can access this
          console. Every action here is recorded to the audit log.
        </p>

        <form className="admin-login-form" onSubmit={handleSubmit}>
          <div className="admin-login-field">
            <label htmlFor="admin-email">Email</label>
            <input
              id="admin-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoComplete="email"
              autoFocus
            />
          </div>

          <div className="admin-login-field">
            <label htmlFor="admin-password">Password</label>
            <input
              id="admin-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              autoComplete="current-password"
            />
          </div>

          {error && <div className="admin-login-error">{error}</div>}

          <button type="submit" className="admin-btn admin-btn-primary" disabled={loading}>
            {loading ? 'Signing in…' : 'Sign in to admin console'}
          </button>
        </form>
      </div>
    </div>
  );
}
