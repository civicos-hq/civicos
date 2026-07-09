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
};

export default sidebars;
