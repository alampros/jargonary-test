import { lazy, Suspense } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import '@fontsource/source-serif-4/latin-600.css'
import '@fontsource/source-serif-4/latin-700.css'
import '@fontsource/dm-sans/latin-400.css'
import '@fontsource/dm-sans/latin-500.css'
import '@fontsource/dm-sans/latin-400-italic.css'
import './styles.css'
import { PublicApp } from './PublicApp'
import type { AdminPage, PublicPage } from './types'

const AdminApp = lazy(() => import('./AdminApp'))
const root = document.getElementById('root')
if (!root) throw new Error('Missing application root')
const data = JSON.parse(root.dataset.bootstrap || '{}') as PublicPage | AdminPage
// Initial HTML contains readable definitions and metadata before this bundle runs.
// Vite may reevaluate this entry when a lazy module becomes a Refresh boundary.
// Reuse its root so React retains ownership of the DOM across those updates.
const reactRoot: Root = import.meta.hot?.data.reactRoot ?? createRoot(root)
if (import.meta.hot) import.meta.hot.data.reactRoot = reactRoot
reactRoot.render(
  data.mode === 'admin' ? (
    <Suspense fallback={<p className="loading">Opening operator console…</p>}>
      <AdminApp data={data} />
    </Suspense>
  ) : (
    <PublicApp data={data as PublicPage} />
  ),
)
