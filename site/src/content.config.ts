import { defineCollection } from 'astro:content'
import { i18nLoader } from '@astrojs/starlight/loaders'
import { docsSchema, i18nSchema } from '@astrojs/starlight/schema'
import { glob } from 'astro/loaders'

export const collections = {
  docs: defineCollection({
    loader: glob({ pattern: '**/*.md', base: './.content' }),
    schema: docsSchema(),
  }),
  i18n: defineCollection({ loader: i18nLoader(), schema: i18nSchema() }),
}
