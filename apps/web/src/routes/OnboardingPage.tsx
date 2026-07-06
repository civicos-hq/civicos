import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { ApiResponse, Community } from '@civicos/types';
import { ArrowLeft, ArrowRight, Check, MapPin, Search, Users } from 'lucide-react';
import { api } from '../lib/api';
import { useMe } from '../hooks/useMe';
import { NIGERIAN_STATES, lgasFor } from '../data/nigeria';
import { LanguageSwitcher } from '../components/LanguageSwitcher';

type Step = 'state' | 'lga' | 'community';

export function OnboardingPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data: me } = useMe();

  const [step, setStep] = useState<Step>('state');
  const [state, setState] = useState<string>('');
  const [lga, setLGA] = useState<string>('');

  // Users who already picked a community shouldn't land on the wizard — they
  // can still reach /profile if they want to change it.
  useEffect(() => {
    if (me?.activeCommunityId) navigate('/discover', { replace: true });
  }, [me, navigate]);

  const communitiesQuery = useQuery({
    queryKey: ['communities', 'byLocation', state, lga],
    enabled: step === 'community' && Boolean(state && lga),
    queryFn: async () => {
      const res = await api.get<ApiResponse<{ communities: Community[] }>>('/api/v1/communities', {
        params: { state, lga },
      });
      return res.data.data.communities;
    },
  });

  const joinMutation = useMutation({
    mutationFn: async (communityId: string) => {
      await api.post('/api/v1/auth/me/community', { communityId });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['me'] });
      navigate('/discover', { replace: true });
    },
  });

  function skip() {
    navigate('/discover', { replace: true });
  }

  return (
    <section className="onboarding-shell">
      <div className="auth-pulse auth-pulse-left" aria-hidden="true" />
      <div className="auth-pulse auth-pulse-right" aria-hidden="true" />

      <div className="auth-lang">
        <LanguageSwitcher />
      </div>

      <div className="onboarding-card">
        <header className="onboarding-header">
          <p className="onboarding-eyebrow">{t('auth.onboarding.eyebrow')}</p>
          <h1 className="onboarding-title">{t('auth.onboarding.title')}</h1>
          <p className="onboarding-description">{t('auth.onboarding.description')}</p>
          <div
            className="onboarding-stepper"
            role="progressbar"
            aria-valuemin={1}
            aria-valuemax={3}
            aria-valuenow={step === 'state' ? 1 : step === 'lga' ? 2 : 3}
          >
            <span className={`onboarding-step-dot ${step !== 'state' ? 'is-done' : 'is-active'}`}>
              1
            </span>
            <span className="onboarding-step-line" />
            <span
              className={`onboarding-step-dot ${step === 'community' ? 'is-done' : step === 'lga' ? 'is-active' : ''}`}
            >
              2
            </span>
            <span className="onboarding-step-line" />
            <span className={`onboarding-step-dot ${step === 'community' ? 'is-active' : ''}`}>
              3
            </span>
          </div>
          <p className="onboarding-step-label">
            {t('auth.onboarding.step', { n: step === 'state' ? 1 : step === 'lga' ? 2 : 3 })}
          </p>
        </header>

        {step === 'state' && (
          <StatePicker
            selected={state}
            onSelect={(name) => {
              setState(name);
              setLGA('');
              setStep('lga');
            }}
            onSkip={skip}
          />
        )}

        {step === 'lga' && (
          <LGAPicker
            state={state}
            selected={lga}
            onBack={() => setStep('state')}
            onSelect={(name) => {
              setLGA(name);
              setStep('community');
            }}
          />
        )}

        {step === 'community' && (
          <CommunityPicker
            state={state}
            lga={lga}
            communities={communitiesQuery.data ?? []}
            loading={communitiesQuery.isLoading}
            onBack={() => setStep('lga')}
            onSkip={skip}
            onJoin={(id) => joinMutation.mutate(id)}
            joining={joinMutation.isPending}
          />
        )}
      </div>
    </section>
  );
}

// ─── Step 1 ────────────────────────────────────────────────────────────

function StatePicker({
  selected,
  onSelect,
  onSkip,
}: {
  selected: string;
  onSelect: (name: string) => void;
  onSkip: () => void;
}) {
  const { t } = useTranslation();
  const [query, setQuery] = useState('');
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return NIGERIAN_STATES;
    return NIGERIAN_STATES.filter((s) => s.name.toLowerCase().includes(q));
  }, [query]);

  return (
    <>
      <h2 className="onboarding-question">{t('auth.onboarding.stateTitle')}</h2>
      <p className="onboarding-hint">{t('auth.onboarding.stateHint')}</p>

      <label className="onboarding-search" htmlFor="state-search">
        <Search className="h-4 w-4" aria-hidden="true" />
        <input
          id="state-search"
          type="search"
          placeholder={t('auth.onboarding.statePlaceholder')}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          autoFocus
        />
      </label>

      <ul className="onboarding-list" role="listbox">
        {filtered.map((s) => (
          <li key={s.code}>
            <button
              type="button"
              role="option"
              aria-selected={s.name === selected}
              className={`onboarding-option ${s.name === selected ? 'is-selected' : ''}`}
              onClick={() => onSelect(s.name)}
            >
              <span className="onboarding-option-label">{s.name}</span>
              <span className="onboarding-option-meta">{s.lgas.length} LGAs</span>
              {s.name === selected && (
                <Check className="onboarding-option-check h-4 w-4" aria-hidden="true" />
              )}
            </button>
          </li>
        ))}
      </ul>

      <div className="onboarding-actions">
        <button type="button" className="onboarding-skip" onClick={onSkip}>
          {t('auth.onboarding.skip')}
        </button>
      </div>
    </>
  );
}

// ─── Step 2 ────────────────────────────────────────────────────────────

function LGAPicker({
  state,
  selected,
  onSelect,
  onBack,
}: {
  state: string;
  selected: string;
  onSelect: (name: string) => void;
  onBack: () => void;
}) {
  const { t } = useTranslation();
  const [query, setQuery] = useState('');
  const lgas = lgasFor(state);
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return lgas;
    return lgas.filter((l) => l.toLowerCase().includes(q));
  }, [query, lgas]);

  return (
    <>
      <h2 className="onboarding-question">{t('auth.onboarding.lgaTitle')}</h2>
      <p className="onboarding-hint">{t('auth.onboarding.lgaHint', { state })}</p>

      <label className="onboarding-search" htmlFor="lga-search">
        <Search className="h-4 w-4" aria-hidden="true" />
        <input
          id="lga-search"
          type="search"
          placeholder={t('auth.onboarding.lgaPlaceholder')}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          autoFocus
        />
      </label>

      <ul className="onboarding-list" role="listbox">
        {filtered.map((name) => (
          <li key={name}>
            <button
              type="button"
              role="option"
              aria-selected={name === selected}
              className={`onboarding-option ${name === selected ? 'is-selected' : ''}`}
              onClick={() => onSelect(name)}
            >
              <MapPin className="onboarding-option-icon h-4 w-4" aria-hidden="true" />
              <span className="onboarding-option-label">{name}</span>
              {name === selected && (
                <Check className="onboarding-option-check h-4 w-4" aria-hidden="true" />
              )}
            </button>
          </li>
        ))}
      </ul>

      <div className="onboarding-actions">
        <button type="button" className="onboarding-back" onClick={onBack}>
          <ArrowLeft className="h-4 w-4" />
          {t('auth.onboarding.back')}
        </button>
      </div>
    </>
  );
}

// ─── Step 3 ────────────────────────────────────────────────────────────

function CommunityPicker({
  state,
  lga,
  communities,
  loading,
  onBack,
  onSkip,
  onJoin,
  joining,
}: {
  state: string;
  lga: string;
  communities: Community[];
  loading: boolean;
  onBack: () => void;
  onSkip: () => void;
  onJoin: (id: string) => void;
  joining: boolean;
}) {
  const { t } = useTranslation();
  const [selected, setSelected] = useState<string>('');

  return (
    <>
      <h2 className="onboarding-question">{t('auth.onboarding.communityTitle')}</h2>
      <p className="onboarding-hint">{t('auth.onboarding.communityHint', { state, lga })}</p>

      {loading && <p className="onboarding-loading">{t('auth.onboarding.communityLoading')}</p>}

      {!loading && communities.length === 0 && (
        <div className="onboarding-empty">
          <Users className="h-6 w-6" aria-hidden="true" />
          <div>
            <p className="onboarding-empty-title">{t('auth.onboarding.communityEmptyTitle')}</p>
            <p className="onboarding-empty-sub">
              {t('auth.onboarding.communityEmptySub', { state, lga })}
            </p>
          </div>
        </div>
      )}

      {!loading && communities.length > 0 && (
        <ul className="onboarding-list" role="listbox">
          {communities.map((c) => (
            <li key={c.id}>
              <button
                type="button"
                role="option"
                aria-selected={c.id === selected}
                className={`onboarding-community ${c.id === selected ? 'is-selected' : ''}`}
                onClick={() => setSelected(c.id)}
              >
                <div className="onboarding-community-body">
                  <span className="onboarding-community-name">{c.name}</span>
                  {c.description && (
                    <span className="onboarding-community-sub">{c.description}</span>
                  )}
                  <span className="onboarding-community-meta">
                    {c.memberCount} {c.memberCount === 1 ? 'member' : 'members'}
                  </span>
                </div>
                {c.id === selected && (
                  <Check className="onboarding-option-check h-4 w-4" aria-hidden="true" />
                )}
              </button>
            </li>
          ))}
        </ul>
      )}

      <div className="onboarding-actions onboarding-actions--split">
        <button type="button" className="onboarding-back" onClick={onBack}>
          <ArrowLeft className="h-4 w-4" />
          {t('auth.onboarding.back')}
        </button>
        <div className="onboarding-actions-right">
          <button type="button" className="onboarding-skip" onClick={onSkip}>
            {t('auth.onboarding.skip')}
          </button>
          {communities.length > 0 && (
            <button
              type="button"
              className="onboarding-primary"
              disabled={!selected || joining}
              onClick={() => selected && onJoin(selected)}
            >
              {joining ? t('auth.onboarding.joining') : t('auth.onboarding.join')}
              <ArrowRight className="h-4 w-4" />
            </button>
          )}
        </div>
      </div>
    </>
  );
}
