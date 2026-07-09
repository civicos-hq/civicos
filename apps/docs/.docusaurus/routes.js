import React from 'react';
import ComponentCreator from '@docusaurus/ComponentCreator';

export default [
  {
    path: '/__docusaurus/debug',
    component: ComponentCreator('/__docusaurus/debug', '5ff'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/config',
    component: ComponentCreator('/__docusaurus/debug/config', '5ba'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/content',
    component: ComponentCreator('/__docusaurus/debug/content', 'a2b'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/globalData',
    component: ComponentCreator('/__docusaurus/debug/globalData', 'c3c'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/metadata',
    component: ComponentCreator('/__docusaurus/debug/metadata', '156'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/registry',
    component: ComponentCreator('/__docusaurus/debug/registry', '88c'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/routes',
    component: ComponentCreator('/__docusaurus/debug/routes', '000'),
    exact: true
  },
  {
    path: '/',
    component: ComponentCreator('/', 'c0d'),
    routes: [
      {
        path: '/',
        component: ComponentCreator('/', '2f4'),
        routes: [
          {
            path: '/',
            component: ComponentCreator('/', '476'),
            routes: [
              {
                path: '/citizens/notifications',
                component: ComponentCreator('/citizens/notifications', '8d5'),
                exact: true,
                sidebar: "userGuide"
              },
              {
                path: '/citizens/report-issue',
                component: ComponentCreator('/citizens/report-issue', 'de3'),
                exact: true,
                sidebar: "userGuide"
              },
              {
                path: '/citizens/voting',
                component: ComponentCreator('/citizens/voting', '28d'),
                exact: true,
                sidebar: "userGuide"
              },
              {
                path: '/getting-started/create-account',
                component: ComponentCreator('/getting-started/create-account', 'a13'),
                exact: true,
                sidebar: "userGuide"
              },
              {
                path: '/getting-started/join-community',
                component: ComponentCreator('/getting-started/join-community', '5ff'),
                exact: true,
                sidebar: "userGuide"
              },
              {
                path: '/organizations/managing-organizations',
                component: ComponentCreator('/organizations/managing-organizations', 'fc5'),
                exact: true,
                sidebar: "userGuide"
              },
              {
                path: '/representatives/dashboard',
                component: ComponentCreator('/representatives/dashboard', '483'),
                exact: true,
                sidebar: "userGuide"
              },
              {
                path: '/',
                component: ComponentCreator('/', 'b1e'),
                exact: true,
                sidebar: "userGuide"
              }
            ]
          }
        ]
      }
    ]
  },
  {
    path: '*',
    component: ComponentCreator('*'),
  },
];
