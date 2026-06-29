export function NotificationsPage() {
  const notifications = [
    {
      title: 'Issue status updated',
      detail: 'Streetlight outage moved to IN_PROGRESS',
      when: '12 min ago',
    },
    {
      title: 'Petition milestone reached',
      detail: 'Road repair petition crossed 1,000 signatures',
      when: '1 hr ago',
    },
    {
      title: 'Representative response posted',
      detail: 'Local council replied on drainage concerns',
      when: 'Yesterday',
    },
  ];

  return (
    <section className="space-y-6">
      <header className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <p className="text-xs font-semibold uppercase tracking-[0.14em] text-civic-700">
          Notification Center
        </p>
        <h1 className="mt-2 text-3xl font-semibold text-slate-900">Civic activity feed</h1>
      </header>

      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h2 className="text-lg font-semibold text-slate-900">Latest updates</h2>
        <div className="mt-4 grid gap-3">
          {notifications.map((note) => (
            <article
              key={note.title}
              className="rounded-xl border border-slate-200 bg-slate-50/70 p-4"
            >
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <h3 className="font-semibold text-slate-900">{note.title}</h3>
                  <p className="mt-1 text-sm text-slate-600">{note.detail}</p>
                </div>
                <span className="text-xs font-medium text-slate-500">{note.when}</span>
              </div>
            </article>
          ))}
        </div>
      </section>
    </section>
  );
}
