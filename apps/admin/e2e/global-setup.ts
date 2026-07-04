import { execFileSync } from 'node:child_process';
import { purgeAdminE2eData, jwtSecret } from './fixtures/db';

// Sanity-checks that the backend stack is up and that we can craft
// tokens before we run any spec.
export default async function globalSetup() {
  const services = [
    { name: 'gateway', url: 'http://localhost:3000/health' },
    { name: 'identity-service', url: 'http://localhost:3001/health' },
  ];

  for (const svc of services) {
    try {
      const res = await fetch(svc.url);
      if (!res.ok) throw new Error(`status ${res.status}`);
      const body = (await res.json()) as { status?: string };
      if (body.status !== 'ok') throw new Error(`bad status: ${JSON.stringify(body)}`);
    } catch (err) {
      throw new Error(
        `[admin e2e preflight] ${svc.name} not reachable at ${svc.url}: ${(err as Error).message}\n` +
          `Start the backend services before running the admin e2e suite.`,
      );
    }
  }

  try {
    jwtSecret();
  } catch (err) {
    throw new Error(`[admin e2e preflight] ${(err as Error).message}`);
  }

  try {
    execFileSync('docker', ['exec', 'civicos_postgres', 'pg_isready', '-U', 'civicos'], {
      stdio: 'ignore',
    });
  } catch {
    throw new Error(`[admin e2e preflight] cannot reach civicos_postgres`);
  }

  purgeAdminE2eData();
}
