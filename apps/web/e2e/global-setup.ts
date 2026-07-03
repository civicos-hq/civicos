import { execFileSync } from 'node:child_process';
import { purgeE2eData, jwtSecret } from './fixtures/db';

// Runs once before the whole suite. Sanity-checks that the backend stack
// is reachable and that the JWT_SECRET can be read; if either fails we
// bail fast with an actionable error message instead of letting every
// spec time out mid-navigation.
export default async function globalSetup() {
  const services: Array<{ name: string; url: string }> = [
    { name: 'gateway', url: 'http://localhost:3000/health' },
    { name: 'identity-service', url: 'http://localhost:3001/health' },
    { name: 'community-service', url: 'http://localhost:3002/health' },
    { name: 'organization-service', url: 'http://localhost:3003/health' },
  ];

  for (const svc of services) {
    try {
      const res = await fetch(svc.url);
      if (!res.ok) throw new Error(`status ${res.status}`);
      const body = (await res.json()) as { status?: string };
      if (body.status !== 'ok') throw new Error(`bad status: ${JSON.stringify(body)}`);
    } catch (err) {
      throw new Error(
        `[e2e preflight] ${svc.name} not reachable at ${svc.url}: ${(err as Error).message}\n` +
          `Start the services before running the e2e suite.`,
      );
    }
  }

  // JWT_SECRET must be present — the fixtures need it to craft tokens.
  try {
    jwtSecret();
  } catch (err) {
    throw new Error(`[e2e preflight] ${(err as Error).message}`);
  }

  // docker must be able to reach the Postgres container.
  try {
    execFileSync('docker', ['exec', 'civicos_postgres', 'pg_isready', '-U', 'civicos'], {
      stdio: 'ignore',
    });
  } catch {
    throw new Error(`[e2e preflight] cannot reach civicos_postgres container`);
  }

  // Fresh slate — nuke any e2e-leftovers from a previous run.
  purgeE2eData();
}
