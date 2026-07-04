import { purgeAdminE2eData } from './fixtures/db';

export default async function globalTeardown() {
  purgeAdminE2eData();
}
