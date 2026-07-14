import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Flag } from 'lucide-react';
import { ReportModal, type ReportableType } from './ReportModal';

interface Props {
  contentType: ReportableType;
  contentId: string;
  size?: 'sm' | 'md';
  className?: string;
}

// Small trigger button that spawns the ReportModal. Kept intentionally
// tiny so it can live alongside comment/announcement metadata without
// dominating the row.
export function ReportButton({ contentType, contentId, size = 'sm', className = '' }: Props) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

  const isSmall = size === 'sm';
  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-label={t('report.buttonAria')}
        className={`inline-flex items-center gap-1 text-slate-500 dark:text-slate-300 hover:text-slate-800 dark:hover:text-slate-200 transition ${
          isSmall ? 'text-xs' : 'text-sm'
        } ${className}`}
      >
        <Flag className={isSmall ? 'h-3 w-3' : 'h-3.5 w-3.5'} aria-hidden="true" />
        <span>{t('report.buttonLabel')}</span>
      </button>
      {open && (
        <ReportModal
          contentType={contentType}
          contentId={contentId}
          onClose={() => setOpen(false)}
        />
      )}
    </>
  );
}
