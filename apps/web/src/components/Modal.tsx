import { useEffect, type ReactNode } from 'react';
import { createPortal } from 'react-dom';

/**
 * Sizes for the panel. `lg` is the default and matches every modal that
 * existed before this prop; `xl` is for forms with enough fields that a
 * single narrow column becomes unreadably tall.
 */
type Size = 'lg' | 'xl';

const widths: Record<Size, string> = {
  lg: 'max-w-lg',
  xl: 'max-w-2xl',
};

export function Modal({
  title,
  onClose,
  children,
  size = 'lg',
}: {
  title: string;
  onClose: () => void;
  children: ReactNode;
  size?: Size;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  // Rendered through a PORTAL, into document.body.
  //
  // Without it the overlay is `position: fixed` inside whatever subtree
  // happened to render the modal — and any ancestor with a transform, filter
  // or will-change becomes its containing block instead of the viewport. On
  // the org dashboard that opened the dialog ~500px down the page with half
  // of it off-screen, and made it lurch as the page moved underneath. A
  // portal lifts it out of that subtree so `fixed` means fixed.
  return createPortal(
    <div
      // Anchored to the TOP, and the overlay itself never scrolls.
      //
      // A centred panel re-centres on every height change, so a validation
      // message appearing as you type yanked the whole dialog upward. A
      // scrollable overlay was just as bad: focusing a field scrolled it to
      // reveal the field, moving the panel under the cursor. The panel is
      // capped and scrolls its own body instead, so once open its position
      // does not change.
      className="fixed inset-0 z-50 flex items-start justify-center bg-slate-900/40 p-4 sm:p-6"
      onClick={onClose}
      role="presentation"
    >
      <div
        role="dialog"
        aria-modal="true"
        className={`flex max-h-[calc(100vh-3rem)] w-full ${widths[size]} flex-col rounded-2xl bg-white shadow-xl dark:bg-slate-900`}
        onClick={(e) => e.stopPropagation()}
      >
        {/* The header stays put while the body scrolls, so the title and the
            close button remain reachable at any scroll position. */}
        <div className="flex items-start justify-between gap-4 border-b border-slate-200 px-6 py-4 dark:border-slate-700">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">{title}</h2>
          <button
            type="button"
            className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-400"
            onClick={onClose}
            aria-label="Close"
          >
            ✕
          </button>
        </div>
        <div className="overflow-y-auto px-6 py-5">{children}</div>
      </div>
    </div>,
    document.body,
  );
}
