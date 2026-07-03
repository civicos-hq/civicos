import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { MapPin } from 'lucide-react';

/**
 * The shared "join a community first" notice. Previously this pattern
 * appeared four times across Discover / Issues / Petitions / Reps with
 * four slightly different implementations. Now they all render this
 * single amber card so the message reads as one voice.
 *
 * Children carry the page-specific action verb ("...to start reporting
 * issues", "...and this feed will start sorting by proximity", etc).
 * Use the exported `CommunityGateLink` inside those children whenever
 * a link is needed — it takes the visual weight that used to be spelled
 * out with underline classes in every page.
 */
export function CommunityGate({ children }: { children: ReactNode }) {
  return (
    <div className="community-gate" role="status">
      <span className="community-gate-icon" aria-hidden="true">
        <MapPin className="h-4 w-4" />
      </span>
      <p className="community-gate-body">{children}</p>
    </div>
  );
}

/** The bolded link inside a CommunityGate. Always points to /community. */
export function CommunityGateLink({ children }: { children: ReactNode }) {
  return (
    <Link to="/community" className="community-gate-link">
      {children}
    </Link>
  );
}
