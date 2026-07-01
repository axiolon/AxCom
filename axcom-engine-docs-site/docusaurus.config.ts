/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import { themes as prismThemes } from "prism-react-renderer";
import type { Config } from "@docusaurus/types";
import type * as Preset from "@docusaurus/preset-classic";

const config: Config = {
  title: "AxCom",
  tagline: "High-performance transactional commerce engine built in Go.",
  favicon: "img/favicon.ico",

  url: "https://docs.AxCom.dev",
  baseUrl: "/axcom/",

  organizationName: "axiolon",
  projectName: "AxCom",

  onBrokenLinks: "throw",
  markdown: {
    mermaid: true,
    hooks: {
      onBrokenMarkdownLinks: "warn",
    },
  },

  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },

  themes: ["@docusaurus/theme-mermaid"],

  presets: [
    [
      "redocusaurus",
      {
        specs: [
          {
            id: "axcom-engine",
            spec: "../axcom-backend/engineering/api/openapi.yaml",
            route: "/api/",
          },
        ],
        theme: {
          primaryColor: "#1E4F8B",
        },
      },
    ],
    [
      "classic",
      {
        docs: {
          path: "../docs",
          routeBasePath: "/docs",
          sidebarPath: "./sidebars.ts",
          editUrl: "https://github.com/axiolon/AxCom/edit/main/",
          showLastUpdateTime: true,
          showLastUpdateAuthor: true,
        },
        blog: false,
        theme: {
          customCss: "./src/css/custom.css",
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    colorMode: {
      defaultMode: "light",
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: "AxCom",
      logo: {
        alt: "AxCom Logo",
        src: "img/favicon.svg",
      },
      items: [
        {
          type: "docSidebar",
          sidebarId: "docsSidebar",
          position: "left",
          label: "Docs",
        },
        {
          to: "/api/",
          label: "API Reference",
          position: "left",
        },
        {
          href: "https://github.com/axiolon/AxCom",
          label: "GitHub",
          position: "right",
        },
      ],
      hideOnScroll: true,
    },
    footer: {
      style: "dark",
      links: [
        {
          title: "Documentation",
          items: [
            { label: "Getting Started", to: "/docs/getting-started/intro" },
          ],
        },
        {
          title: "Project",
          items: [
            {
              label: "GitHub",
              href: "https://github.com/axiolon/AxCom",
            },
            {
              label: "Issues",
              href: "https://github.com/axiolon/AxCom/issues",
            },
          ],
        },
        {
          title: "Axiolon Labs",
          items: [
            {
              label: "Website",
              href: "https://axiolonlabs.dev",
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Axiolon Labs. Apache-2.0 Licensed.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.vsDark,
      additionalLanguages: ["go", "bash", "yaml", "json"],
    },
    docs: {
      sidebar: {
        hideable: true,
        autoCollapseCategories: true,
      },
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
