import { lazy, Suspense } from 'react'
import { createRoot } from 'react-dom/client'
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
createRoot(root).render(
  data.mode === 'admin' ? (
    <Suspense fallback={<p className="loading">Opening operator console…</p>}>
      <AdminApp data={data} />
    </Suspense>
  ) : (
    <PublicApp data={data as PublicPage} />
  ),
)
