import { Button } from '@civicos/ui';

export function IssuesPage() {
  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Issues</h1>
        <Button size="sm">+ Report Issue</Button>
      </div>
      <p className="text-gray-500">Issue list coming soon.</p>
    </div>
  );
}
