import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import type { Community } from '@civicos/types';
import { ArrowLeft, ArrowRight, Check, Home, MapPin, Search, Users, X } from 'lucide-react';
import { api } from '../lib/api';
import { useMe } from '../hooks/useMe';
import { useCommunities } from '../hooks/useCommunities';
import { NIGERIAN_STATES, lgasFor } from '../data/nigeria';
import { LanguageSwitcher } from '../components/LanguageSwitcher';

type Step = 'pick' | 'home';

const RESULTS_PER_PAGE = 25;

/**
 * Community onboarding.
 *
 * This used to be a strict State → LGA → Community drill-down, which made
 * a whole class of community unreachable: a student looking for their
 * university had to already know which local government area the campus
 * sits in. Abuja alone spreads its universities across three area
 * councils — University of Abuja in Gwagwalada, Nile and Baze in the
 * Municipal Area Council — so the wrong first guess simply showed an
 * empty list, with no hint that the place existed at all.
 *
 * So search leads and the drill-down is the fallback, not the gate. Two
 * further changes follow from the same observation:
 *
 *   • Several communities can be joined at once, because the honest
 *     answer is usually more than one ("I live here, I study there").
 *   • Which one is *home* is asked explicitly rather than inferred from
 *     join order, because home decides where a citizen may raise issues
 *     and petitions and is rate-limited to one change per 30 days.
 */
export function OnboardingPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data: me } = useMe();

  const [step, setStep] = useState<Step>('pick');
  const [query, setQuery] = useState('');
  const [state, setState] = useState('');
  const [lga, setLGA] = useState('');
  const [selected, setSelected] = useState<Community[]>([]);
  const [primaryId, setPrimaryId] = useState('');
  const [page, setPage] = useState(0);

  // Users who already have a community shouldn't land on the wizard — they
  // can still change things from /community.
  useEffect(() => {
    if (me?.activeCommunityId) navigate('/discover', { replace: true });
  }, [me, navigate]);

  // Reset paging whenever the filters change, so page 3 of an old search
  // never carries over into a new one and shows nothing.
  useEffect(() => {
    setPage(0);
  }, [query, state, lga]);

  const communitiesQuery = useCommunities({
    q: query,
    state,
    lga,
    limit: RESULTS_PER_PAGE,
    offset: page * RESULTS_PER_PAGE,
  });

  const results = communitiesQuery.data?.communities ?? [];
  const total = communitiesQuery.data?.total ?? 0;
  const hasMore = (page + 1) * RESULTS_PER_PAGE < total;

  const selectedIds = useMemo(() => new Set(selected.map((c) => c.id)), [selected]);

  function toggle(community: Community) {
    setSelected((prev) => {
      const exists = prev.some((c) => c.id === community.id);
      const next = exists ? prev.filter((c) => c.id !== community.id) : [...prev, community];
      // Keep the nominated home valid: if it was just removed, fall back
      // to the first remaining pick rather than submitting an orphan.
      setPrimaryId((current) => {
        if (next.length === 0) return '';
        if (next.some((c) => c.id === current)) return current;
        return next[0].id;
      });
      return next;
    });
  }

  const joinMutation = useMutation({
    mutationFn: async () => {
      await api.post('/api/v1/auth/me/communities', {
        communityIds: selected.map((c) => c.id),
        primaryCommunityId: primaryId,
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['me'] });
      navigate('/discover', { replace: true });
    },
  });

  function skip() {
    navigate('/discover', { replace: true });
  }

  const stepIndex = step === 'pick' ? 1 : 2;

  return (
    <section className="onboarding-shell">
      <div className="auth-pulse auth-pulse-left" aria-hidden="true" />
      <div className="auth-pulse auth-pulse-right" aria-hidden="true" />

      <Link to="/" className="auth-home" aria-label={t('common.backToHome')}>
        {t('common.backToHomeShort')}
      </Link>

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
            aria-valuemax={2}
            aria-valuenow={stepIndex}
          >
            <span className={`onboarding-step-dot ${step === 'pick' ? 'is-active' : 'is-done'}`}>
              1
            </span>
            <span className="onboarding-step-line" />
            <span className={`onboarding-step-dot ${step === 'home' ? 'is-active' : ''}`}>2</span>
          </div>
          <p className="onboarding-step-label">
            {t('auth.onboarding.stepOf', { n: stepIndex, total: 2 })}
          </p>
        </header>

        {step === 'pick' ? (
          <PickStep
            query={query}
            onQueryChange={setQuery}
            state={state}
            lga={lga}
            onStateChange={(next) => {
              setState(next);
              setLGA('');
            }}
            onLGAChange={setLGA}
            results={results}
            total={total}
            loading={communitiesQuery.isLoading}
            selectedIds={selectedIds}
            selected={selected}
            onToggle={toggle}
            page={page}
            hasMore={hasMore}
            onPage={setPage}
            onSkip={skip}
            onContinue={() => setStep('home')}
          />
        ) : (
          <HomeStep
            selected={selected}
            primaryId={primaryId}
            onPrimaryChange={setPrimaryId}
            onBack={() => setStep('pick')}
            onFinish={() => joinMutation.mutate()}
            saving={joinMutation.isPending}
            failed={joinMutation.isError}
          />
        )}
      </div>
    </section>
  );
}

// ─── Step 1 — find and pick ────────────────────────────────────────────

function PickStep({
  query,
  onQueryChange,
  state,
  lga,
  onStateChange,
  onLGAChange,
  results,
  total,
  loading,
  selectedIds,
  selected,
  onToggle,
  page,
  hasMore,
  onPage,
  onSkip,
  onContinue,
}: {
  query: string;
  onQueryChange: (v: string) => void;
  state: string;
  lga: string;
  onStateChange: (v: string) => void;
  onLGAChange: (v: string) => void;
  results: Community[];
  total: number;
  loading: boolean;
  selectedIds: Set<string>;
  selected: Community[];
  onToggle: (c: Community) => void;
  page: number;
  hasMore: boolean;
  onPage: (updater: (p: number) => number) => void;
  onSkip: () => void;
  onContinue: () => void;
}) {
  const { t } = useTranslation();
  const [browsing, setBrowsing] = useState(false);
  const lgas = useMemo(() => (state ? lgasFor(state) : []), [state]);

  return (
    <>
      <h2 className="onboarding-question">{t('auth.onboarding.pickTitle')}</h2>
      <p className="onboarding-hint">{t('auth.onboarding.pickHint')}</p>

      <label className="onboarding-search" htmlFor="community-search">
        <Search className="h-4 w-4" aria-hidden="true" />
        <input
          id="community-search"
          type="search"
          placeholder={t('auth.onboarding.searchPlaceholder')}
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
          autoFocus
        />
      </label>

      {/* The old drill-down, demoted to an optional filter. Kept because
          it is genuinely the better tool when you don't know the name of
          anything and just want to see what exists near you. */}
      <div className="mt-3">
        <button
          type="button"
          className="text-sm font-medium text-emerald-700 underline-offset-2 hover:underline dark:text-emerald-400"
          onClick={() => setBrowsing((v) => !v)}
          aria-expanded={browsing}
        >
          {browsing ? t('auth.onboarding.hideBrowse') : t('auth.onboarding.showBrowse')}
        </button>

        {browsing && (
          <div className="mt-3 grid gap-3 sm:grid-cols-2">
            <label className="flex flex-col gap-1 text-sm">
              <span className="font-medium text-slate-700 dark:text-slate-200">
                {t('auth.onboarding.stateLabel')}
              </span>
              <select
                className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-slate-600 dark:bg-slate-800"
                value={state}
                onChange={(e) => onStateChange(e.target.value)}
              >
                <option value="">{t('auth.onboarding.anyState')}</option>
                {NIGERIAN_STATES.map((s) => (
                  <option key={s.code} value={s.name}>
                    {s.name}
                  </option>
                ))}
              </select>
            </label>

            <label className="flex flex-col gap-1 text-sm">
              <span className="font-medium text-slate-700 dark:text-slate-200">
                {t('auth.onboarding.lgaLabel')}
              </span>
              <select
                className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm disabled:opacity-50 dark:border-slate-600 dark:bg-slate-800"
                value={lga}
                onChange={(e) => onLGAChange(e.target.value)}
                disabled={!state}
              >
                <option value="">{t('auth.onboarding.anyLGA')}</option>
                {lgas.map((name) => (
                  <option key={name} value={name}>
                    {name}
                  </option>
                ))}
              </select>
            </label>
          </div>
        )}
      </div>

      {selected.length > 0 && (
        <ul className="mt-4 flex flex-wrap gap-2" aria-label={t('auth.onboarding.selectedLabel')}>
          {selected.map((c) => (
            <li key={c.id}>
              <button
                type="button"
                className="inline-flex items-center gap-1.5 rounded-full bg-emerald-600 px-3 py-1 text-sm font-medium text-white hover:bg-emerald-700"
                onClick={() => onToggle(c)}
              >
                {c.name}
                <X className="h-3.5 w-3.5" aria-hidden="true" />
                <span className="sr-only">{t('auth.onboarding.removeSelection')}</span>
              </button>
            </li>
          ))}
        </ul>
      )}

      {loading && <p className="onboarding-loading">{t('auth.onboarding.communityLoading')}</p>}

      {!loading && results.length === 0 && (
        <div className="onboarding-empty">
          <Users className="h-6 w-6" aria-hidden="true" />
          <div>
            <p className="onboarding-empty-title">{t('auth.onboarding.noResultsTitle')}</p>
            <p className="onboarding-empty-sub">{t('auth.onboarding.noResultsSub')}</p>
          </div>
        </div>
      )}

      {!loading && results.length > 0 && (
        <>
          <p className="mt-4 text-xs text-slate-500 dark:text-slate-400">
            {t('auth.onboarding.resultCount', { count: total })}
          </p>
          <ul className="onboarding-list">
            {results.map((c) => {
              const isSelected = selectedIds.has(c.id);
              return (
                <li key={c.id}>
                  <button
                    type="button"
                    role="checkbox"
                    aria-checked={isSelected}
                    className={`onboarding-community ${isSelected ? 'is-selected' : ''}`}
                    onClick={() => onToggle(c)}
                  >
                    <div className="onboarding-community-body">
                      <span className="onboarding-community-name">{c.name}</span>
                      <span className="onboarding-community-sub">
                        {c.lga}, {c.state}
                      </span>
                      <span className="onboarding-community-meta">
                        {t('common.memberCount', { count: c.memberCount ?? 0 })}
                      </span>
                    </div>
                    {isSelected && (
                      <Check className="onboarding-option-check h-4 w-4" aria-hidden="true" />
                    )}
                  </button>
                </li>
              );
            })}
          </ul>

          {(page > 0 || hasMore) && (
            <div className="mt-3 flex items-center justify-between text-sm">
              <button
                type="button"
                className="onboarding-back disabled:opacity-40"
                onClick={() => onPage((p) => Math.max(0, p - 1))}
                disabled={page === 0}
              >
                <ArrowLeft className="h-4 w-4" />
                {t('common.previous')}
              </button>
              <button
                type="button"
                className="onboarding-back disabled:opacity-40"
                onClick={() => onPage((p) => p + 1)}
                disabled={!hasMore}
              >
                {t('common.next')}
                <ArrowRight className="h-4 w-4" />
              </button>
            </div>
          )}
        </>
      )}

      <div className="onboarding-actions onboarding-actions--split">
        <button type="button" className="onboarding-skip" onClick={onSkip}>
          {t('auth.onboarding.skip')}
        </button>
        <div className="onboarding-actions-right">
          <button
            type="button"
            className="onboarding-primary"
            disabled={selected.length === 0}
            onClick={onContinue}
          >
            {t('auth.onboarding.continueWith', { count: selected.length })}
            <ArrowRight className="h-4 w-4" />
          </button>
        </div>
      </div>
    </>
  );
}

// ─── Step 2 — nominate a home community ────────────────────────────────

function HomeStep({
  selected,
  primaryId,
  onPrimaryChange,
  onBack,
  onFinish,
  saving,
  failed,
}: {
  selected: Community[];
  primaryId: string;
  onPrimaryChange: (id: string) => void;
  onBack: () => void;
  onFinish: () => void;
  saving: boolean;
  failed: boolean;
}) {
  const { t } = useTranslation();

  return (
    <>
      <h2 className="onboarding-question">{t('auth.onboarding.homeTitle')}</h2>
      <p className="onboarding-hint">{t('auth.onboarding.homeHint')}</p>

      <ul className="onboarding-list" role="radiogroup" aria-label={t('auth.onboarding.homeTitle')}>
        {selected.map((c) => {
          const isPrimary = c.id === primaryId;
          return (
            <li key={c.id}>
              <button
                type="button"
                role="radio"
                aria-checked={isPrimary}
                className={`onboarding-community ${isPrimary ? 'is-selected' : ''}`}
                onClick={() => onPrimaryChange(c.id)}
              >
                <div className="onboarding-community-body">
                  <span className="onboarding-community-name">
                    <Home className="mr-1.5 inline h-3.5 w-3.5" aria-hidden="true" />
                    {c.name}
                  </span>
                  <span className="onboarding-community-sub">
                    {c.lga}, {c.state}
                  </span>
                </div>
                {isPrimary && (
                  <Check className="onboarding-option-check h-4 w-4" aria-hidden="true" />
                )}
              </button>
            </li>
          );
        })}
      </ul>

      {/* Stating the cooldown up front. It is a 30-day commitment and the
          old flow never mentioned it — users discovered the limit only
          when they tried to undo a choice they never knew they made. */}
      <p className="mt-3 flex items-start gap-2 rounded-lg bg-amber-50 p-3 text-xs text-amber-900 dark:bg-amber-950/40 dark:text-amber-200">
        <MapPin className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
        {t('auth.onboarding.homeCooldownNote')}
      </p>

      {failed && (
        <p className="mt-3 text-sm text-red-600 dark:text-red-400">
          {t('auth.onboarding.joinError')}
        </p>
      )}

      <div className="onboarding-actions onboarding-actions--split">
        <button type="button" className="onboarding-back" onClick={onBack}>
          <ArrowLeft className="h-4 w-4" />
          {t('auth.onboarding.back')}
        </button>
        <div className="onboarding-actions-right">
          <button
            type="button"
            className="onboarding-primary"
            disabled={!primaryId || saving}
            onClick={onFinish}
          >
            {saving ? t('auth.onboarding.joining') : t('auth.onboarding.finish')}
            <ArrowRight className="h-4 w-4" />
          </button>
        </div>
      </div>
    </>
  );
}
