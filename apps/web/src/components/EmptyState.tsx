import type { ReactNode } from 'react';

/**
 * The one empty-state shape every list uses. Before this: seven ad-hoc
 * variants — plain <p>, dashed-border card, muted italic, etc. Now all
 * empty states read as "this section has nothing yet" with the same
 * dashed card + optional icon + optional action.
 *
 *   - Icon is optional but recommended (lucide-react is fine; anything
 *     rendered inline works).
 *   - Body is optional — sometimes just a headline is enough.
 *   - Action is optional — a Button, Link, or anything.
 *
 * We keep the surrounding container out of scope: whatever wrapping the
 * page already has (space-y, grid, etc.) still owns positioning. The
 * component only styles what's inside the empty card.
 */
export interface EmptyStateProps {
  icon?: ReactNode;
  title: ReactNode;
  body?: ReactNode;
  action?: ReactNode;
}

export function EmptyState({ icon, title, body, action }: EmptyStateProps) {
  return (
    <div className="empty-state" role="status">
      {icon && (
        <span className="empty-state-icon" aria-hidden="true">
          {icon}
        </span>
      )}
      <p className="empty-state-title">{title}</p>
      {body && <p className="empty-state-body">{body}</p>}
      {action && <div className="empty-state-action">{action}</div>}
    </div>
  );
}
