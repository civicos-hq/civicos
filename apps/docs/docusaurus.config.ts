import type { Config } from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';
import { themes as prismThemes } from 'prism-react-renderer';

const config: Config = {
  title: 'CivicOS',
  tagline: 'How to use CivicOS',
  favicon: 'img/favicon.ico',

  url: 'https://docs.civicos.ng',
  baseUrl: '/',

  organizationName: 'civicos',
  projectName: 'civicos',

  onBrokenLinks: 'warn',
  onBrokenMarkdownLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: '/',
          editUrl: 'https://github.com/civicos/civicos/tree/main/apps/docs/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  // Local, index-based search — no external service, no signup. Builds
  // an index at compile time and ships it with the static bundle.
  themes: [
    [
      require.resolve('@easyops-cn/docusaurus-search-local'),
      {
        hashed: true,
        indexBlog: false,
        docsRouteBasePath: '/',
        highlightSearchTermsOnTargetPage: true,
      },
    ],
  ],

  themeConfig: {
    colorMode: {
      defaultMode: 'light',
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'CivicOS',
      logo: {
        alt: 'CivicOS',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'userGuide',
          position: 'left',
          label: 'User Guide',
        },
        {
          type: 'docSidebar',
          sidebarId: 'developerGuide',
          position: 'left',
          label: 'Developer Guide',
        },
        {
          href: 'http://localhost:3000/docs',
          label: 'API Reference',
          position: 'right',
        },
        {
          href: 'https://github.com/civicos/civicos',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Learn',
          items: [
            { label: 'Create an account', to: '/getting-started/create-account' },
            { label: 'Report an issue', to: '/citizens/report-issue' },
            { label: 'Voting', to: '/citizens/voting' },
          ],
        },
        {
          title: 'Roles',
          items: [
            { label: 'For citizens', to: '/citizens/report-issue' },
            { label: 'For organizations', to: '/organizations/managing-organizations' },
            { label: 'For representatives', to: '/representatives/dashboard' },
          ],
        },
        {
          title: 'Developers',
          items: [
            { label: 'API Reference (Swagger)', href: 'http://localhost:3000/docs' },
            { label: 'GitHub', href: 'https://github.com/civicos/civicos' },
          ],
        },
      ],
      copyright: `© ${new Date().getFullYear()} CivicOS. Built for democratic participation.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
