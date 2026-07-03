import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Search, Loader2 } from 'lucide-react';
import type { Petition, Representative } from '@civicos/types';
import { useSearch } from '../../hooks/useSearch';
import { useEnumLabels } from '../../hooks/useEnumLabels';

export function SearchBar() {
  const { t } = useTranslation();
  const enums = useEnumLabels();
  const [query, setQuery] = useState('');
  const [open, setOpen] = useState(false);
  const navigate = useNavigate();
  const wrapRef = useRef<HTMLDivElement | null>(null);

  const { data, isFetching, enabled, debouncedQuery } = useSearch(query);

  // Close on outside click.
  useEffect(() => {
    function onDocClick(e: MouseEvent) {
      if (!wrapRef.current) return;
      if (!wrapRef.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener('mousedown', onDocClick);
    return () => document.removeEventListener('mousedown', onDocClick);
  }, []);

  function go(path: string) {
    setOpen(false);
    setQuery('');
    navigate(path);
  }

  const total = data.issues.length + data.petitions.length + data.representatives.length;
  const showDropdown = open && enabled;
  const showEmpty = showDropdown && !isFetching && total === 0 && debouncedQuery.length >= 2;

  return (
    <div ref={wrapRef} className="relative max-w-xl flex-1">
      <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
      <input
        type="search"
        value={query}
        placeholder={t('search.placeholder')}
        aria-label={t('search.placeholder')}
        className="dashboard-search"
        onChange={(e) => {
          setQuery(e.target.value);
          setOpen(true);
        }}
        onFocus={() => setOpen(true)}
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            setOpen(false);
            (e.target as HTMLInputElement).blur();
          }
        }}
      />
      {isFetching && enabled && (
        <Loader2 className="pointer-events-none absolute right-3 top-1/2 h-4 w-4 -translate-y-1/2 animate-spin text-slate-400" />
      )}

      {showDropdown && (
        <div className="absolute left-0 right-0 top-[calc(100%+0.5rem)] z-20 max-h-[28rem] overflow-y-auto rounded-2xl border border-slate-200 bg-white shadow-lg">
          {showEmpty && (
            <p className="px-4 py-6 text-center text-sm text-slate-600">
              {t('search.empty', { query: debouncedQuery })}
            </p>
          )}

          {data.issues.length > 0 && (
            <Section title={t('search.groups.issues')}>
              {data.issues.map((it) => (
                <ResultRow
                  key={it.id}
                  primary={it.title}
                  secondary={`${enums.issueStatus(it.status)} · ${t('issuesPage.meta.upvotes', {
                    count: it.upvoteCount,
                  })}`}
                  onClick={() => go(`/issues/${it.id}`)}
                />
              ))}
            </Section>
          )}

          {data.petitions.length > 0 && (
            <Section title={t('search.groups.petitions')}>
              {data.petitions.map((p: Petition) => (
                <ResultRow
                  key={p.id}
                  primary={p.title}
                  secondary={`${t('discoverPage.meta.signaturesOf', {
                    signatures: p.signatureCount,
                    goal: p.goal,
                  })} · ${enums.petitionStatus(p.status)}`}
                  onClick={() => go(`/petitions/${p.id}`)}
                />
              ))}
            </Section>
          )}

          {data.representatives.length > 0 && (
            <Section title={t('search.groups.representatives')}>
              {data.representatives.map((r: Representative) => (
                <ResultRow
                  key={r.id}
                  primary={r.name}
                  secondary={`${r.position} · ${r.constituency}`}
                  onClick={() => go(`/representatives/${r.id}`)}
                />
              ))}
            </Section>
          )}
        </div>
      )}
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="border-b border-slate-100 last:border-b-0">
      <p className="px-4 pb-1 pt-3 text-[10px] font-semibold uppercase tracking-[0.14em] text-slate-400">
        {title}
      </p>
      <div className="pb-2">{children}</div>
    </div>
  );
}

function ResultRow({
  primary,
  secondary,
  onClick,
}: {
  primary: string;
  secondary: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex w-full flex-col items-start gap-0.5 px-4 py-2 text-left hover:bg-slate-50"
    >
      <span className="line-clamp-1 text-sm font-medium text-slate-900">{primary}</span>
      <span className="line-clamp-1 text-xs text-slate-600">{secondary}</span>
    </button>
  );
}
