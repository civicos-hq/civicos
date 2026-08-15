/**
 * Mock content for the pre-launch test window.
 *
 * ── What this deliberately avoids ────────────────────────────────────
 * Nothing here names a real person, alleges wrongdoing by anyone, or
 * quotes an identifiable official. The issues are the kind of thing a
 * resident actually reports — a burst main, a dark street, a clinic
 * short of staff — because the point is to see whether the product
 * works, not to invent a scandal.
 *
 * Organizations and representatives carry a visible label by default
 * (see DEMO_LABEL in seed-demo.mjs). A fictional "Senator" attached to a
 * real constituency on a public site is the one piece of this that could
 * genuinely mislead someone, and a label costs nothing.
 *
 * ── Every account is disposable ──────────────────────────────────────
 * All of it is removed by the pre-launch database reset. Nothing here
 * should ever be visible to a real citizen.
 */

/** Communities the seeder ensures exist. Coordinates are real. */
export const COMMUNITIES = [
  {
    name: 'Makurdi',
    slug: 'demo-makurdi',
    state: 'Benue',
    lga: 'Makurdi',
    latitude: 7.7322,
    longitude: 8.5391,
    description: 'Benue State capital, on the south bank of the River Benue.',
  },
  {
    name: 'Lokoja',
    slug: 'demo-lokoja',
    state: 'Kogi',
    lga: 'Lokoja',
    latitude: 7.8023,
    longitude: 6.7333,
    description: 'Kogi State capital, at the confluence of the Niger and Benue.',
  },
  {
    name: 'Ikeja',
    slug: 'demo-ikeja',
    state: 'Lagos',
    lga: 'Ikeja',
    latitude: 6.6018,
    longitude: 3.3515,
    description: 'Lagos State capital.',
  },
  {
    name: 'Gwagwalada',
    slug: 'demo-gwagwalada',
    state: 'FCT',
    lga: 'Gwagwalada',
    latitude: 8.9434,
    longitude: 7.0797,
    description: 'Area council in the Federal Capital Territory.',
  },
];

/**
 * Citizen personas. Names are ordinary Nigerian names; the demo email
 * domain is what marks them, so the app looks the way it will on launch.
 */
export const CITIZENS = [
  { key: 'amaka', name: 'Amaka Eze', community: 'demo-makurdi' },
  { key: 'ibrahim', name: 'Ibrahim Sule', community: 'demo-makurdi' },
  { key: 'tunde', name: 'Tunde Bakare', community: 'demo-ikeja' },
  { key: 'ngozi', name: 'Ngozi Okonkwo', community: 'demo-ikeja' },
  { key: 'fatima', name: 'Fatima Bello', community: 'demo-lokoja' },
  { key: 'emeka', name: 'Emeka Nwachukwu', community: 'demo-gwagwalada' },
];

/**
 * Issues, spread across categories and the status lifecycle so the
 * filters and the status timeline both have something to show.
 * `upvoters` and `commenters` reference citizen keys.
 */
export const ISSUES = [
  {
    community: 'demo-makurdi',
    author: 'amaka',
    title: 'Burst water main flooding Wurukum roundabout',
    description:
      'The main on the Wurukum approach has been leaking for six days and the road is now under water at the roundabout. Okada riders are going around it through the market, which is dangerous at that junction.',
    category: 'UTILITIES',
    location: 'Wurukum roundabout',
    status: 'UNDER_REVIEW',
    upvoters: ['ibrahim', 'fatima', 'emeka'],
    comments: [
      { author: 'ibrahim', body: 'Still leaking this morning. The whole lane is impassable now.' },
    ],
  },
  {
    community: 'demo-makurdi',
    author: 'ibrahim',
    title: 'No streetlights on High Level road for three weeks',
    description:
      'The lights between the junction and the bridge have been off since the last storm. It is completely dark by 7pm and people are walking in the road.',
    category: 'INFRASTRUCTURE',
    location: 'High Level',
    status: 'IN_PROGRESS',
    upvoters: ['amaka', 'emeka'],
    comments: [],
  },
  {
    community: 'demo-makurdi',
    author: 'amaka',
    title: 'Drainage blocked ahead of the rains',
    description:
      'The channel behind the market has silted up completely. Last year the same blockage put water into six shops. It needs clearing before the rains start properly.',
    category: 'ENVIRONMENT',
    location: 'Modern Market',
    status: 'OPEN',
    upvoters: ['ibrahim'],
    comments: [],
  },
  {
    community: 'demo-ikeja',
    author: 'tunde',
    title: 'Traffic light at Allen junction has been out for a month',
    description:
      'Both signals are dead. Traffic wardens are managing it during the day but there is nobody after 6pm and the junction backs up badly.',
    category: 'TRANSPORT',
    location: 'Allen Avenue junction',
    status: 'RESOLVED',
    upvoters: ['ngozi', 'tunde'],
    comments: [{ author: 'ngozi', body: 'Fixed as of Tuesday — thank you to whoever chased it.' }],
  },
  {
    community: 'demo-ikeja',
    author: 'ngozi',
    title: 'Primary health centre has no night staff',
    description:
      'The centre closes at 4pm. Anyone who needs care in the evening has to travel, which for people without transport means waiting until morning.',
    category: 'HEALTH',
    location: 'Oregun',
    status: 'OPEN',
    upvoters: ['tunde'],
    comments: [],
  },
  {
    community: 'demo-ikeja',
    author: 'tunde',
    title: 'Refuse not collected on our street for two weeks',
    description:
      'Collection used to be Tuesdays. Nothing for a fortnight and the bins are overflowing into the road.',
    category: 'ENVIRONMENT',
    location: 'Opebi',
    status: 'UNDER_REVIEW',
    upvoters: ['ngozi'],
    comments: [],
  },
  {
    community: 'demo-lokoja',
    author: 'fatima',
    title: 'River bank erosion behind the secondary school',
    description:
      'The bank has moved several metres closer to the school fence since last season. The classrooms nearest the fence are the ones being used by the junior students.',
    category: 'ENVIRONMENT',
    location: 'Adankolo',
    status: 'UNDER_REVIEW',
    upvoters: ['amaka', 'emeka'],
    comments: [],
  },
  {
    community: 'demo-lokoja',
    author: 'fatima',
    title: 'Power outages lasting most of the day',
    description:
      'We have had supply for perhaps four hours a day for the past two weeks. Small businesses on this street are running generators the whole time.',
    category: 'UTILITIES',
    location: 'Ganaja',
    status: 'OPEN',
    upvoters: [],
    comments: [],
  },
  {
    community: 'demo-gwagwalada',
    author: 'emeka',
    title: 'Borehole at the community centre has stopped working',
    description:
      'The pump has been down since last month. People are walking to the next ward for water, which is about forty minutes each way.',
    category: 'UTILITIES',
    location: 'Phase 2',
    status: 'IN_PROGRESS',
    upvoters: ['amaka', 'fatima', 'tunde'],
    comments: [{ author: 'amaka', body: 'Same problem in our area last year — it was the pump.' }],
  },
  {
    community: 'demo-gwagwalada',
    author: 'emeka',
    title: 'Road to the clinic impassable after rain',
    description:
      'The last stretch is untarred and turns to mud. Vehicles cannot get through, so anyone being taken to the clinic is carried the last part.',
    category: 'INFRASTRUCTURE',
    location: 'Clinic road',
    status: 'OPEN',
    upvoters: ['fatima'],
    comments: [],
  },
];

/** Petitions at varying signature levels so progress bars differ. */
export const PETITIONS = [
  {
    community: 'demo-makurdi',
    author: 'amaka',
    title: 'Clear the market drainage before the rainy season',
    description:
      'We are asking for the drainage channel behind Modern Market to be cleared before the rains. It flooded six shops last year and the same blockage is back. Traders have offered to help with labour if the equipment is provided.',
    goal: 500,
    signers: ['ibrahim', 'fatima', 'emeka', 'tunde'],
  },
  {
    community: 'demo-ikeja',
    author: 'ngozi',
    title: 'Extend primary health centre hours to 10pm',
    description:
      'The centre closing at 4pm leaves the whole ward without care in the evening. We are asking for staffing to cover until 10pm, which is when most emergencies here actually happen.',
    goal: 1000,
    signers: ['tunde', 'amaka'],
  },
  {
    community: 'demo-lokoja',
    author: 'fatima',
    title: 'Reinforce the river bank behind Adankolo secondary school',
    description:
      'The bank has moved several metres toward the school fence in one season. We are asking for the bank to be reinforced before the classrooms nearest the fence become unsafe.',
    goal: 300,
    signers: ['amaka', 'ibrahim', 'emeka', 'ngozi', 'tunde'],
  },
  {
    community: 'demo-gwagwalada',
    author: 'emeka',
    title: 'Repair the community centre borehole',
    description:
      'The pump has been down for a month and people are walking forty minutes each way for water. We are asking for it to be repaired, and for a maintenance arrangement so it does not happen again.',
    goal: 250,
    signers: ['fatima'],
  },
];

/**
 * Representatives. Labelled by default — see the note at the top of this
 * file about fictional officials on a public site.
 */
export const REPRESENTATIVES = [
  {
    key: 'rep-benue',
    name: 'Adaeze Nwosu',
    title: 'Hon.',
    position: 'Member, State House of Assembly',
    constituency: 'Makurdi North',
    community: 'demo-makurdi',
    party: 'Independent',
    bio: 'Sample representative profile for pre-launch testing.',
    announcements: [
      {
        title: 'Drainage clearing begins Monday at Modern Market',
        body: 'Following reports raised here, contractors will begin clearing the market channel on Monday. Work is expected to take a week. Traders on the affected row have been notified directly.',
        publish: true,
      },
      {
        title: 'Ward meeting on flood preparation — Saturday',
        body: 'An open meeting on flood preparation will be held at the community hall on Saturday morning. Anyone in the ward is welcome.',
        publish: true,
      },
    ],
  },
  {
    key: 'rep-fct',
    name: 'Musa Danjuma',
    title: 'Hon.',
    position: 'Councillor',
    constituency: 'Gwagwalada Ward 3',
    community: 'demo-gwagwalada',
    party: 'Independent',
    bio: 'Sample representative profile for pre-launch testing.',
    announcements: [
      {
        title: 'Borehole repair scheduled',
        body: 'A technician has assessed the community centre borehole and the pump is being replaced. I will post again when the work is done.',
        publish: true,
      },
    ],
  },
];

/**
 * Organizations, created through the real application-and-approval flow.
 * `fundingReady` marks the ones the seeder will make campaign-eligible.
 */
export const ORGANIZATIONS = [
  {
    key: 'water-board',
    name: 'Benue State Water Board',
    slug: 'demo-benue-water-board',
    kind: 'UTILITY',
    jurisdiction: 'STATE',
    state: 'Benue',
    lga: 'Makurdi',
    description: 'Sample utility organization for pre-launch testing.',
    ownerName: 'Grace Terhemba',
    fundingReady: false,
    announcements: [
      {
        title: 'Wurukum main repair — supply interruption Thursday',
        body: 'Repairs to the burst main at Wurukum will interrupt supply across the area on Thursday between 6am and 4pm. Tankers will be stationed at the roundabout and at the market.',
        publish: true,
      },
      {
        title: 'Reporting a leak',
        body: 'Leaks can be reported here on CivicOS and will be assigned to the district team. Please include the nearest landmark — it is the single most useful detail for finding a leak quickly.',
        publish: true,
      },
    ],
    projects: [
      {
        title: 'Wurukum distribution main replacement',
        description:
          'Replacing 1.2km of ageing distribution main between the treatment works and Wurukum roundabout, which is the source of the repeated bursts on that stretch.',
        status: 'ACTIVE',
        budgetNaira: 42000000,
        community: 'demo-makurdi',
      },
      {
        title: 'Ward 4 metering pilot',
        description:
          'Installing meters on 300 connections to measure actual distribution losses before committing to a wider programme.',
        status: 'PLANNED',
        budgetNaira: 8500000,
        community: 'demo-makurdi',
      },
    ],
    consultations: [
      {
        title: 'Where should the next public standpipes go?',
        summary:
          'We have funding for six standpipes and want to place them where they are most needed rather than where they are easiest to install.',
        description:
          'Six public standpipes are funded for this financial year. Rather than placing them where installation is cheapest, we want to hear where people are actually walking furthest for water. Responses close at the end of the month and the chosen locations will be published here.',
        publish: true,
        questions: [
          {
            prompt: 'Which area needs a standpipe most?',
            type: 'SINGLE_CHOICE',
            options: ['Wurukum', 'High Level', 'Modern Market', 'North Bank'],
          },
          {
            prompt: 'How far do you currently walk for water?',
            type: 'SINGLE_CHOICE',
            options: ['Under 5 minutes', '5–15 minutes', '15–30 minutes', 'Over 30 minutes'],
          },
          { prompt: 'Anything else we should know?', type: 'FREE_TEXT' },
        ],
      },
    ],
    campaigns: [],
  },
  {
    key: 'relief-ngo',
    name: 'Benue Flood Relief Initiative',
    slug: 'demo-benue-flood-relief',
    kind: 'NGO',
    jurisdiction: 'STATE',
    state: 'Benue',
    lga: 'Makurdi',
    description: 'Sample NGO for pre-launch testing of Community Funding.',
    ownerName: 'Samuel Iorver',
    fundingReady: true,
    announcements: [
      {
        title: 'Distribution completed in North Bank',
        body: 'Relief materials reached 240 households in North Bank this week. A full breakdown of what was distributed and what it cost is on the campaign page.',
        publish: true,
      },
    ],
    projects: [
      {
        title: 'Emergency shelter materials store',
        description:
          'Maintaining a pre-positioned store of tarpaulins, mats and water containers so distribution can begin within 48 hours of a flood rather than after procurement.',
        status: 'ACTIVE',
        budgetNaira: 15000000,
        community: 'demo-makurdi',
      },
    ],
    consultations: [],
    campaigns: [
      {
        title: 'Emergency relief for North Bank flood households',
        summary:
          'Tarpaulins, sleeping mats and water containers for households displaced by the last flood.',
        description:
          'The last flood displaced households across North Bank. This campaign funds shelter materials and clean water containers for the families still without them. Distribution is done through the ward committee, and every disbursement is published on this page as it happens.',
        category: 'EMERGENCY_RELIEF',
        goalNaira: 3500000,
        isEmergency: true,
        community: 'demo-makurdi',
        milestones: [
          { title: 'First 100 households', targetNaira: 1200000 },
          { title: 'Remaining 140 households', targetNaira: 2300000 },
        ],
      },
    ],
  },
  {
    key: 'lagos-agency',
    name: 'Ikeja Environmental Task Force',
    slug: 'demo-ikeja-environment',
    kind: 'AGENCY',
    jurisdiction: 'LGA',
    state: 'Lagos',
    lga: 'Ikeja',
    description: 'Sample local agency for pre-launch testing.',
    ownerName: 'Bisi Adeyemi',
    fundingReady: false,
    announcements: [
      {
        title: 'Refuse collection returning to schedule',
        body: 'Two additional vehicles are back in service from Monday and collection returns to the normal Tuesday schedule across Opebi and Oregun. Missed collections can be reported here.',
        publish: true,
      },
    ],
    projects: [],
    consultations: [
      {
        title: 'What should collection day be?',
        summary:
          'We are reviewing collection days across the LGA and want to know what actually works for residents.',
        description:
          'Collection is currently Tuesday across the whole LGA, which does not suit every street. We are reviewing the schedule and want to know which day and time actually work for residents before we change anything.',
        publish: true,
        questions: [
          {
            prompt: 'Which day suits you best?',
            type: 'SINGLE_CHOICE',
            options: ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'],
          },
          {
            prompt: 'What time of day?',
            type: 'SINGLE_CHOICE',
            options: ['Morning', 'Afternoon', 'Evening'],
          },
        ],
      },
    ],
    campaigns: [],
  },
];
