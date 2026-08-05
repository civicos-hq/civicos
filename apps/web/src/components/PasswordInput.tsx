import { useState, type ChangeEvent } from 'react';
import { Eye, EyeOff } from 'lucide-react';
import { useTranslation } from 'react-i18next';

interface PasswordInputProps {
  id: string;
  value: string;
  onChange: (event: ChangeEvent<HTMLInputElement>) => void;
  placeholder?: string;
  /** 'current-password' on sign-in, 'new-password' on register/reset. */
  autoComplete?: string;
  required?: boolean;
  minLength?: number;
}

/**
 * Password field with a reveal toggle.
 *
 * Typing a password blind is the single most error-prone moment in the auth
 * flow — on registration a typo isn't caught until the reset email arrives.
 *
 * Notes on the details that matter:
 * - `type="button"` is load-bearing. A bare <button> inside a <form> defaults
 *   to type="submit", so the toggle would submit the form instead.
 * - Swapping the `type` attribute keeps the same DOM node, so focus and the
 *   browser's password-manager binding survive the toggle.
 * - The state is per-field and always starts masked, so a reveal never
 *   persists across navigations or leaks into a second field.
 */
export function PasswordInput({
  id,
  value,
  onChange,
  placeholder,
  autoComplete,
  required,
  minLength,
}: PasswordInputProps) {
  const { t } = useTranslation();
  const [revealed, setRevealed] = useState(false);

  return (
    <div className="auth-password">
      <input
        id={id}
        type={revealed ? 'text' : 'password'}
        className="auth-input auth-password-input"
        placeholder={placeholder}
        value={value}
        onChange={onChange}
        autoComplete={autoComplete}
        required={required}
        minLength={minLength}
      />
      <button
        type="button"
        className="auth-password-toggle"
        onClick={() => setRevealed((current) => !current)}
        aria-label={revealed ? t('auth.fields.hidePassword') : t('auth.fields.showPassword')}
        aria-pressed={revealed}
        aria-controls={id}
      >
        {revealed ? (
          <EyeOff className="h-4 w-4" aria-hidden="true" />
        ) : (
          <Eye className="h-4 w-4" aria-hidden="true" />
        )}
      </button>
    </div>
  );
}
