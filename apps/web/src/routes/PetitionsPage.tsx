import { Button } from '@civicos/ui';

export function PetitionsPage() {
  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Petitions</h1>
        <Button size="sm">+ New Petition</Button>
      </div>
      <p className="text-gray-500">Petitions coming soon.</p>
    </div>
  );
}
