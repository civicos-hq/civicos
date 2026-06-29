export function ProfilePage() {
  return (
    <section className="space-y-6">
      <header className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <p className="text-xs font-semibold uppercase tracking-[0.14em] text-civic-700">
          Citizen Profile
        </p>
        <h1 className="mt-2 text-3xl font-semibold text-slate-900">Manage your civic identity</h1>
      </header>

      <div className="grid gap-4 lg:grid-cols-[1.2fr,0.8fr]">
        <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-slate-900">Profile details</h2>
          <div className="mt-4 grid gap-3 sm:grid-cols-2">
            <label className="text-sm text-slate-700">
              Full name
              <input
                className="mt-1.5 w-full rounded-lg border border-slate-200 px-3 py-2"
                placeholder="Ada Okonkwo"
              />
            </label>
            <label className="text-sm text-slate-700">
              Email
              <input
                className="mt-1.5 w-full rounded-lg border border-slate-200 px-3 py-2"
                placeholder="ada@civicos.org"
              />
            </label>
          </div>
          <button className="mt-4 rounded-lg bg-civic-700 px-4 py-2 text-sm font-semibold text-white hover:bg-civic-600">
            Save changes
          </button>
        </section>

        <aside className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-slate-900">Participation snapshot</h2>
          <div className="mt-4 space-y-2 text-sm text-slate-700">
            <p>
              <span className="font-semibold">Issues reported:</span> 4
            </p>
            <p>
              <span className="font-semibold">Petitions signed:</span> 9
            </p>
            <p>
              <span className="font-semibold">Community:</span> Ikeja Central
            </p>
          </div>
        </aside>
      </div>
    </section>
  );
}
