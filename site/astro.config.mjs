import { unified } from '@astrojs/markdown-remark'
import starlight from '@astrojs/starlight'
import { defineConfig } from 'astro/config'
import remarkLinks from './remark-links.mjs'

const url = new URL(process.env.PAGES_BASE_URL || 'https://alampros.github.io/jargonary')

export default defineConfig({
  site: url.origin,
  base: url.pathname,
  outDir: '../.cache/pages',
  markdown: { processor: unified({ remarkPlugins: [[remarkLinks, { base: url.pathname }]] }) },
  integrations: [
    starlight({
      title: 'Jargonary',
      description: 'Publish team-owned glossaries and bring their vocabulary together.',
      social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/alampros/jargonary' }],
      customCss: ['./src/styles/custom.css'],
      sidebar: [
        { label: 'Overview', slug: 'index' },
        { label: 'Getting started', slug: 'getting-started' },
        { label: 'Glossary format', slug: 'dictionary-format' },
        { label: 'Configuration and deployment', slug: 'reference' },
        { label: 'Connect an AI agent', slug: 'mcp' },
        { label: 'Branding and themes', slug: 'customization' },
        { label: 'JSON Schema', slug: 'schemas' },
        {
          label: 'Reference',
          collapsed: true,
          items: [
            { label: 'Source settings', slug: 'sources' },
            { label: 'Publishing', slug: 'publishing' },
            { label: 'Status and APIs', slug: 'operations' },
          ],
        },
        {
          label: 'Contributing',
          items: [
            { label: 'Development', slug: 'development' },
            { label: 'SQLite lookup performance', slug: 'sqlite-lookup-performance' },
          ],
        },
      ],
    }),
  ],
})
