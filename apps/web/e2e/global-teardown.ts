import { purgeE2eData } from './fixtures/db';

// Runs after every spec has finished (or errored). Removes anything the
// suite created so successive runs stay clean and the dev DB doesn't fill
// with e2e detritus.
export default async function globalTeardown() {
  purgeE2eData();
}
