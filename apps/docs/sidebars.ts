import type { SidebarsConfig } from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  userGuide: [
    'intro',
    {
      type: 'category',
      label: 'Getting Started',
      collapsed: false,
      items: ['getting-started/create-account', 'getting-started/join-community'],
    },
    {
      type: 'category',
      label: 'For Citizens',
      collapsed: false,
      items: ['citizens/report-issue', 'citizens/voting', 'citizens/notifications'],
    },
    {
      type: 'category',
      label: 'For Organizations',
      collapsed: false,
      items: ['organizations/managing-organizations'],
    },
    {
      type: 'category',
      label: 'For Representatives',
      collapsed: false,
      items: ['representatives/dashboard'],
    },
  ],

  developerGuide: [
    'developer/index',
    {
      type: 'category',
      label: 'Overview',
      collapsed: false,
      items: [
        'developer/overview/architecture',
        'developer/overview/repository-structure',
        'developer/overview/monorepo',
      ],
    },
    {
      type: 'category',
      label: 'Development',
      collapsed: false,
      items: [
        'developer/development/running-locally',
        'developer/development/packages',
        'developer/development/contributing',
      ],
    },
    {
      type: 'category',
      label: 'Services',
      collapsed: false,
      items: [
        'developer/services/api-gateway',
        'developer/services/identity-service',
        'developer/services/community-service',
        'developer/services/organization-service',
      ],
    },
    {
      type: 'category',
      label: 'Backend systems',
      collapsed: false,
      items: ['developer/backend/database', 'developer/backend/events'],
    },
    {
      type: 'category',
      label: 'Operations',
      collapsed: false,
      items: ['developer/operations/deployment'],
    },
  ],
};

export default sidebars;
