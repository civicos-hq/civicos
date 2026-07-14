import { useEffect, useState } from 'react';
import { ChevronLeft, ChevronRight, X } from 'lucide-react';
import { uploadUrl } from '../lib/api';

export function ImageGallery({ filenames, alt }: { filenames: string[]; alt: string }) {
  const [active, setActive] = useState<number | null>(null);

  if (filenames.length === 0) return null;

  return (
    <>
      <div className="mt-4 grid gap-3 sm:grid-cols-2 md:grid-cols-3">
        {filenames.map((filename, i) => (
          <button
            key={filename}
            type="button"
            onClick={() => setActive(i)}
            className="block overflow-hidden rounded-xl ring-1 ring-slate-200 dark:ring-slate-700 transition hover:ring-civic-400 focus:outline-none focus:ring-2 focus:ring-civic-500"
            aria-label={`Open photo ${i + 1}`}
          >
            <img src={uploadUrl(filename)} alt={alt} className="h-40 w-full object-cover" />
          </button>
        ))}
      </div>

      {active !== null && (
        <Lightbox
          filenames={filenames}
          index={active}
          alt={alt}
          onClose={() => setActive(null)}
          onIndex={setActive}
        />
      )}
    </>
  );
}

function Lightbox({
  filenames,
  index,
  alt,
  onClose,
  onIndex,
}: {
  filenames: string[];
  index: number;
  alt: string;
  onClose: () => void;
  onIndex: (i: number) => void;
}) {
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
      else if (e.key === 'ArrowRight') onIndex((index + 1) % filenames.length);
      else if (e.key === 'ArrowLeft') onIndex((index - 1 + filenames.length) % filenames.length);
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [filenames.length, index, onClose, onIndex]);

  const filename = filenames[index]!;
  const hasMultiple = filenames.length > 1;

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-slate-950/80 p-4"
      onClick={onClose}
    >
      <button
        type="button"
        onClick={onClose}
        className="absolute right-4 top-4 rounded-full bg-white/10 p-2 text-white hover:bg-white dark:hover:bg-slate-900/70/20"
        aria-label="Close"
      >
        <X className="h-5 w-5" />
      </button>

      {hasMultiple && (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onIndex((index - 1 + filenames.length) % filenames.length);
          }}
          className="absolute left-4 top-1/2 -translate-y-1/2 rounded-full bg-white/10 p-3 text-white hover:bg-white dark:hover:bg-slate-900/70/20"
          aria-label="Previous photo"
        >
          <ChevronLeft className="h-5 w-5" />
        </button>
      )}

      <img
        src={uploadUrl(filename)}
        alt={alt}
        className="max-h-[85vh] max-w-[85vw] rounded-lg object-contain shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      />

      {hasMultiple && (
        <>
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onIndex((index + 1) % filenames.length);
            }}
            className="absolute right-4 top-1/2 -translate-y-1/2 rounded-full bg-white/10 p-3 text-white hover:bg-white dark:hover:bg-slate-900/70/20"
            aria-label="Next photo"
          >
            <ChevronRight className="h-5 w-5" />
          </button>
          <p className="absolute bottom-4 left-1/2 -translate-x-1/2 rounded-full bg-white/10 px-3 py-1 text-xs font-semibold text-white">
            {index + 1} / {filenames.length}
          </p>
        </>
      )}
    </div>
  );
}
