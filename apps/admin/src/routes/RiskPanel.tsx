import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { AlertTriangle, Info, Sparkles } from 'lucide-react';
import { apiPost } from '../lib/api';

/**
 * CivicAI's read on a campaign, for a reviewer working a queue.
 *
 * Presentation carries as much of the design as the endpoint does, so it is
 * worth being explicit about the choices:
 *
 *  - **It does not run on page load.** A reviewer asks for it. Generating a
 *    fraud reading of every campaign an admin happens to open is both a
 *    waste of quota and a way to make the assessment feel like a property of
 *    the campaign rather than one tool's opinion of it.
 *  - **Every signal shows the innocent explanation next to the concern**, at
 *    equal weight. Most campaigns that look odd are run by people who are bad
 *    at paperwork, and a reviewer who sees only the concern has been handed a
 *    conclusion rather than an observation.
 *  - **The band is styled as a priority, not a verdict.** Even
 *    REVIEW_CLOSELY is amber, not red. Red is for reconciliation drift, where
 *    money has demonstrably gone somewhere it should not have. Nothing here
 *    is demonstrated.
 *  - **The disclaimer is always visible**, not behind a tooltip.
 */

type RiskBand = 'ROUTINE' | 'WORTH_A_LOOK' | 'REVIEW_CLOSELY';

interface RiskSignal {
  concern: string;
  evidence: string;
  innocentExplanation: string;
}

interface Assessment {
  band: RiskBand;
  signals: RiskSignal[];
  whatToCheck: string[];
  confidence: number;
  disclaimer: string;
  model: string;
  generatedAt: string;
  advisory: boolean;
}

const BAND_TEXT: Record<RiskBand, { label: string; tone: string; plain: string }> = {
  ROUTINE: {
    label: 'Routine',
    tone: 'success',
    plain: 'Nothing in the published data stands out. Review in the normal order.',
  },
  WORTH_A_LOOK: {
    label: 'Worth a look',
    tone: 'pending',
    plain: 'One or two things a reviewer should confirm before approving.',
  },
  REVIEW_CLOSELY: {
    label: 'Review closely',
    tone: 'pending',
    plain: 'Several observations that together suggest opening this one first.',
  },
};

export function RiskPanel({ campaignId }: { campaignId: string }) {
  const [assessment, setAssessment] = useState<Assessment | null>(null);
  const [error, setError] = useState('');

  const assess = useMutation({
    mutationFn: () =>
      apiPost<{ assessment: Assessment }>('/api/v1/ai/assess-campaign-risk', { campaignId }),
    onSuccess: (d) => {
      setAssessment(d.assessment);
      setError('');
    },
    onError: (err) => {
      const msg = (err as { response?: { data?: { message?: string } } }).response?.data?.message;
      // Fail open: the reviewer's actual job is unaffected.
      setError(msg ?? 'CivicAI could not assess this campaign. Review it manually.');
    },
  });

  const band = assessment ? BAND_TEXT[assessment.band] : null;

  return (
    <section className="admin-table-shell" style={{ marginBottom: 16 }}>
      <div className="admin-table-toolbar">
        <strong className="text-sm">
          <Sparkles className="mr-1 inline h-4 w-4" aria-hidden="true" />
          CivicAI review notes
        </strong>
        <button
          type="button"
          className="admin-btn admin-btn-sm"
          onClick={() => assess.mutate()}
          disabled={assess.isPending}
        >
          {assess.isPending ? 'Reading…' : assessment ? 'Read again' : 'Ask CivicAI'}
        </button>
      </div>

      <div className="p-4 text-sm">
        {!assessment && !error && !assess.isPending && (
          <p className="text-slate-600">
            An optional second pair of eyes on what this campaign has published. It reads the same
            page you can — no documents, no bank records — and it changes nothing.
          </p>
        )}

        {error && <p className="admin-error">{error}</p>}

        {assessment && (
          <div className="space-y-4">
            <div className={`admin-metric-card admin-metric-card-${band!.tone}`}>
              <div className="admin-metric-label">Suggested review priority</div>
              <div className="admin-metric-value" style={{ fontSize: '1.1rem' }}>
                {band!.label}
              </div>
              <p className="mt-1 text-sm">{band!.plain}</p>
            </div>

            {assessment.signals.length === 0 ? (
              <p className="text-slate-600">
                No specific observations. That is not a clean bill of health — it means nothing in
                the published data stood out.
              </p>
            ) : (
              <div className="space-y-3">
                {assessment.signals.map((s, i) => (
                  <article key={i} className="admin-risk-signal">
                    <p className="admin-risk-concern">
                      <AlertTriangle className="mr-1 inline h-3.5 w-3.5" aria-hidden="true" />
                      {s.concern}
                    </p>
                    <p className="admin-risk-evidence">
                      <span className="admin-risk-label">Based on</span> {s.evidence}
                    </p>
                    {/* Equal weight, deliberately. See the note at the top. */}
                    <p className="admin-risk-innocent">
                      <span className="admin-risk-label">Could equally mean</span>{' '}
                      {s.innocentExplanation}
                    </p>
                  </article>
                ))}
              </div>
            )}

            {assessment.whatToCheck.length > 0 && (
              <div>
                <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                  What a person could check
                </p>
                <ul className="mt-1 list-disc space-y-1 pl-5">
                  {assessment.whatToCheck.map((w, i) => (
                    <li key={i}>{w}</li>
                  ))}
                </ul>
              </div>
            )}

            <p className="admin-risk-disclaimer">
              <Info className="mr-1 inline h-3.5 w-3.5" aria-hidden="true" />
              {assessment.disclaimer}
            </p>
            <p className="mono text-xs text-slate-500">
              {assessment.model} · {new Date(assessment.generatedAt).toLocaleString()} · confidence{' '}
              {Math.round(assessment.confidence * 100)}%
            </p>
          </div>
        )}
      </div>
    </section>
  );
}
