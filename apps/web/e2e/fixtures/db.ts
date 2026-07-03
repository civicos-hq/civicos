import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';

// Reads the shared repo-root .env once — same source of truth the
// go services use. If the tests are moved to CI later, this can flip
// to reading process.env instead.
const ENV_FILE = process.env.CIVICOS_ENV_FILE ?? '/Users/gino/civicos/.env';
const DB_CONTAINER = process.env.DB_CONTAINER ?? 'civicos_postgres';

let jwtSecretCache: string | null = null;
export function jwtSecret(): string {
  if (jwtSecretCache) return jwtSecretCache;
  const line = readFileSync(ENV_FILE, 'utf8')
    .split('\n')
    .find((l) => l.startsWith('JWT_SECRET='));
  if (!line) throw new Error(`JWT_SECRET missing from ${ENV_FILE}`);
  jwtSecretCache = line.slice('JWT_SECRET='.length).replace(/^"/, '').replace(/"$/, '');
  return jwtSecretCache;
}

// Run a single SQL statement against the docker Postgres. Returns the
// first row of unaligned tuple output with newlines and psql command
// tags stripped — the same pattern the API smoke test uses.
export function sql(query: string): string {
  const out = execFileSync(
    'docker',
    ['exec', DB_CONTAINER, 'psql', '-U', 'civicos', '-d', 'civicos', '-tAc', query],
    { encoding: 'utf8' },
  );
  return out.split('\n')[0].replace(/\r/g, '').trim();
}

// Multi-statement variant — useful for cleanup blocks. Ignores stdout.
export function sqlBatch(statements: string): void {
  execFileSync(
    'docker',
    ['exec', '-i', DB_CONTAINER, 'psql', '-U', 'civicos', '-d', 'civicos', '-q'],
    { encoding: 'utf8', input: statements },
  );
}

// Remove every e2e-created record from the DB. Idempotent — relies on
// the FK CASCADE constraints so we only need to delete parent rows.
export function purgeE2eData(): void {
  sqlBatch(`
    DELETE FROM organizations WHERE slug LIKE 'e2e-%';
    DELETE FROM representatives WHERE name LIKE 'E2E %';
    DELETE FROM petitions WHERE title LIKE 'E2E petition%';
    DELETE FROM issues WHERE title LIKE 'E2E issue%';
    DELETE FROM communities WHERE slug LIKE 'e2e-%';
    DELETE FROM users WHERE email LIKE 'e2e-%@civicos.test';
  `);
}
