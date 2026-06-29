import { useState } from 'react';
import { Check, Share2 } from 'lucide-react';
import { Button } from '@civicos/ui';

// ShareButton copies the current page URL to the clipboard, showing a 2-second
// confirmation. On platforms that support the Web Share API (mobile Safari /
// Chrome) it falls back to the native share sheet first.
export function ShareButton({ title }: { title: string }) {
  const [copied, setCopied] = useState(false);

  async function share() {
    const url = window.location.href;
    try {
      if (navigator.share) {
        await navigator.share({ title, url });
        return;
      }
    } catch {
      // User cancelled the native share sheet — fall through to clipboard.
    }
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // ignore
    }
  }

  return (
    <Button variant="secondary" size="sm" onClick={share}>
      {copied ? (
        <>
          <Check className="h-3.5 w-3.5" />
          Copied
        </>
      ) : (
        <>
          <Share2 className="h-3.5 w-3.5" />
          Share
        </>
      )}
    </Button>
  );
}
