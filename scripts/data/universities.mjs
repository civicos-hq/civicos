/**
 * Curated seed list of Nigerian universities as CivicOS communities.
 *
 * ── READ THIS BEFORE SEEDING PRODUCTION ──────────────────────────────
 * The `state` values are reliable. The `lga` values are the weak link:
 * they place each campus in a specific local government area, and a wrong
 * one is not cosmetic — `tierFor` in the Discover feed compares LGA
 * strings to decide what counts as "near you", so a misplaced campus
 * ranks wrongly for everyone in that state. A few here are genuinely
 * contested (multi-campus institutions that straddle an LGA boundary,
 * where the "main" campus is a matter of opinion).
 *
 * Verify against the NUC register before the first production seed:
 * https://www.nuc.edu.ng/nigerian-universities/
 *
 * Every `lga` must match `apps/web/src/data/nigeria.ts` EXACTLY — the
 * seed script fails loudly on any that does not, so a typo cannot reach
 * the database. Note FCT's spelling: 'Municipal Area Council', not
 * 'AMAC' or 'Abuja Municipal'.
 *
 * This list is deliberately not exhaustive. It covers the federal
 * universities plus the FCT private cluster; add state and private
 * institutions as coverage demands, one line each.
 */

/** @typedef {{ name: string, slug: string, state: string, lga: string, description?: string }} SeedCommunity */

/** @type {SeedCommunity[]} */
export const UNIVERSITIES = [
  // ── FCT ────────────────────────────────────────────────────────────
  // The cluster that motivated this work: several universities in one
  // city, spread across three different area councils. None of them was
  // reachable from the old wizard unless you already knew which.
  {
    name: 'University of Abuja',
    slug: 'university-of-abuja',
    state: 'FCT',
    lga: 'Gwagwalada',
  },
  {
    name: 'Nile University of Nigeria',
    slug: 'nile-university-of-nigeria',
    state: 'FCT',
    lga: 'Municipal Area Council',
  },
  { name: 'Baze University', slug: 'baze-university', state: 'FCT', lga: 'Municipal Area Council' },
  {
    name: 'African University of Science and Technology',
    slug: 'african-university-of-science-and-technology',
    state: 'FCT',
    lga: 'Municipal Area Council',
  },
  {
    name: 'National Open University of Nigeria',
    slug: 'national-open-university-of-nigeria',
    state: 'FCT',
    lga: 'Municipal Area Council',
  },
  { name: 'Veritas University', slug: 'veritas-university', state: 'FCT', lga: 'Bwari' },
  {
    name: 'Nigerian Army University Biu',
    slug: 'nigerian-army-university-biu',
    state: 'Borno',
    lga: 'Biu',
  },

  // ── Federal universities, by state ─────────────────────────────────
  {
    name: 'Michael Okpara University of Agriculture',
    slug: 'michael-okpara-university-of-agriculture',
    state: 'Abia',
    lga: 'Ikwuano',
  },
  {
    name: 'Modibbo Adama University',
    slug: 'modibbo-adama-university',
    state: 'Adamawa',
    lga: 'Yola North',
  },
  { name: 'University of Uyo', slug: 'university-of-uyo', state: 'Akwa Ibom', lga: 'Uyo' },
  {
    name: 'Nnamdi Azikiwe University',
    slug: 'nnamdi-azikiwe-university',
    state: 'Anambra',
    lga: 'Awka South',
  },
  {
    name: 'Abubakar Tafawa Balewa University',
    slug: 'abubakar-tafawa-balewa-university',
    state: 'Bauchi',
    lga: 'Bauchi',
  },
  {
    name: 'Federal University Otuoke',
    slug: 'federal-university-otuoke',
    state: 'Bayelsa',
    lga: 'Ogbia',
  },
  {
    name: 'Joseph Sarwuan Tarka University',
    slug: 'joseph-sarwuan-tarka-university',
    state: 'Benue',
    lga: 'Makurdi',
  },
  {
    name: 'University of Maiduguri',
    slug: 'university-of-maiduguri',
    state: 'Borno',
    lga: 'Jere',
  },
  {
    name: 'University of Calabar',
    slug: 'university-of-calabar',
    state: 'Cross River',
    lga: 'Calabar Municipal',
  },
  {
    name: 'Federal University of Petroleum Resources Effurun',
    slug: 'federal-university-of-petroleum-resources-effurun',
    state: 'Delta',
    lga: 'Uvwie',
  },
  {
    name: 'Alex Ekwueme Federal University Ndufu-Alike',
    slug: 'alex-ekwueme-federal-university-ndufu-alike',
    state: 'Ebonyi',
    lga: 'Ikwo',
  },
  { name: 'University of Benin', slug: 'university-of-benin', state: 'Edo', lga: 'Egor' },
  {
    name: 'Federal University Oye-Ekiti',
    slug: 'federal-university-oye-ekiti',
    state: 'Ekiti',
    lga: 'Oye',
  },
  {
    name: 'University of Nigeria, Nsukka',
    slug: 'university-of-nigeria-nsukka',
    state: 'Enugu',
    lga: 'Nsukka',
  },
  {
    name: 'Federal University Kashere',
    slug: 'federal-university-kashere',
    state: 'Gombe',
    lga: 'Akko',
  },
  {
    name: 'Federal University of Technology Owerri',
    slug: 'federal-university-of-technology-owerri',
    state: 'Imo',
    lga: 'Owerri West',
  },
  {
    name: 'Federal University Dutse',
    slug: 'federal-university-dutse',
    state: 'Jigawa',
    lga: 'Dutse',
  },
  {
    name: 'Ahmadu Bello University',
    slug: 'ahmadu-bello-university',
    state: 'Kaduna',
    lga: 'Zaria',
  },
  {
    name: 'Nigerian Defence Academy',
    slug: 'nigerian-defence-academy',
    state: 'Kaduna',
    lga: 'Igabi',
  },
  { name: 'Bayero University Kano', slug: 'bayero-university-kano', state: 'Kano', lga: 'Gwale' },
  {
    name: 'Federal University Dutsin-Ma',
    slug: 'federal-university-dutsin-ma',
    state: 'Katsina',
    lga: 'Dutsin-Ma',
  },
  {
    name: 'Federal University Birnin Kebbi',
    slug: 'federal-university-birnin-kebbi',
    state: 'Kebbi',
    lga: 'Birnin Kebbi',
  },
  {
    name: 'Federal University Lokoja',
    slug: 'federal-university-lokoja',
    state: 'Kogi',
    lga: 'Lokoja',
  },
  {
    name: 'University of Ilorin',
    slug: 'university-of-ilorin',
    state: 'Kwara',
    lga: 'Ilorin South',
  },
  {
    name: 'University of Lagos',
    slug: 'university-of-lagos',
    state: 'Lagos',
    lga: 'Lagos Mainland',
  },
  {
    name: 'Federal University Lafia',
    slug: 'federal-university-lafia',
    state: 'Nasarawa',
    lga: 'Lafia',
  },
  {
    name: 'Federal University of Technology Minna',
    slug: 'federal-university-of-technology-minna',
    state: 'Niger',
    lga: 'Bosso',
  },
  {
    name: 'Federal University of Agriculture Abeokuta',
    slug: 'federal-university-of-agriculture-abeokuta',
    state: 'Ogun',
    lga: 'Odeda',
  },
  {
    name: 'Federal University of Technology Akure',
    slug: 'federal-university-of-technology-akure',
    state: 'Ondo',
    lga: 'Akure South',
  },
  {
    name: 'Obafemi Awolowo University',
    slug: 'obafemi-awolowo-university',
    state: 'Osun',
    lga: 'Ife Central',
  },
  { name: 'University of Ibadan', slug: 'university-of-ibadan', state: 'Oyo', lga: 'Ibadan North' },
  { name: 'University of Jos', slug: 'university-of-jos', state: 'Plateau', lga: 'Jos North' },
  {
    name: 'University of Port Harcourt',
    slug: 'university-of-port-harcourt',
    state: 'Rivers',
    lga: 'Obio/Akpor',
  },
  {
    name: 'Usmanu Danfodiyo University',
    slug: 'usmanu-danfodiyo-university',
    state: 'Sokoto',
    lga: 'Sokoto North',
  },
  {
    name: 'Federal University Wukari',
    slug: 'federal-university-wukari',
    state: 'Taraba',
    lga: 'Wukari',
  },
  {
    name: 'Federal University Gashua',
    slug: 'federal-university-gashua',
    state: 'Yobe',
    lga: 'Bade',
  },
  {
    name: 'Federal University Gusau',
    slug: 'federal-university-gusau',
    state: 'Zamfara',
    lga: 'Gusau',
  },
];
