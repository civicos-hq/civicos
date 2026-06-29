import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { api } from '../lib/api';

export function RegisterPage() {
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setErrorMessage('');
    setIsSubmitting(true);

    try {
      await api.post('/api/v1/auth/register', { name, email, password });
      navigate('/login', { replace: true });
    } catch (err) {
      console.error('Registration error:', err);
      setErrorMessage('Could not create your account. Try a different email or try again shortly.');
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
          <p className="auth-eyebrow">CivicOS Registration</p>
          <h1 className="auth-title">Build your citizen profile.</h1>
          <p className="auth-description">
            Join your local community space, report issues faster, and shape public decisions
            between elections.
          </p>
        </aside>

        <form className="auth-card" onSubmit={onSubmit}>
          <h2 className="auth-card-title">Create account</h2>
          <p className="auth-card-subtitle">It takes less than a minute to get started.</p>

          <label className="auth-label" htmlFor="name">
            Full name
          </label>
          <input
            id="name"
            type="text"
            className="auth-input"
            placeholder="Ada Okonkwo"
            value={name}
            onChange={(event) => setName(event.target.value)}
            autoComplete="name"
            required
            minLength={2}
          />

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
            autoComplete="new-password"
            required
            minLength={8}
          />

          {errorMessage && <p className="auth-error">{errorMessage}</p>}

          <button type="submit" className="auth-submit" disabled={isSubmitting}>
            {isSubmitting ? 'Creating account...' : 'Create account'}
          </button>

          <p className="auth-footer">
            Already have an account?{' '}
            <Link to="/login" className="auth-link">
              Sign in
            </Link>
          </p>
        </form>
      </div>
    </section>
  );
}
