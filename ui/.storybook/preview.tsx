import type { Preview } from '@storybook/react-vite'
import { mswLoader } from 'msw-storybook-addon/csf3'
import '@fontsource/source-serif-4/latin-600.css'
import '@fontsource/source-serif-4/latin-700.css'
import '@fontsource/dm-sans/latin-400.css'
import '@fontsource/dm-sans/latin-500.css'
import '@fontsource/dm-sans/latin-400-italic.css'
import '../src/styles.css'
import '../../internal/themes/default.css'
import { mswHandlers } from './msw-handlers'

const preview: Preview = {
  loaders: [mswLoader()],
  beforeEach({ msw }) {
    msw.use(...mswHandlers)
    // The operator view reads its initial tab from the URL fragment.
    history.replaceState(null, '', `${location.pathname}${location.search}`)
  },
  parameters: {
    layout: 'fullscreen',
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },

    a11y: {
      // 'todo' - show a11y violations in the test UI only
      // 'error' - fail CI on a11y violations
      // 'off' - skip a11y checks entirely
      test: 'todo',
    },
  },
}

export default preview
