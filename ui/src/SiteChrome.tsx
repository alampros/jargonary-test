import { BookOpenIcon, GithubLogoIcon, NotePencilIcon } from '@phosphor-icons/react'

function ProjectLinks({ label }: { label: string }) {
  return (
    <nav className="project-links" aria-label={label}>
      <a href="https://alampros.github.io/jargonary/">
        <BookOpenIcon size={20} aria-hidden />
        Docs
      </a>
      <a href="https://github.com/alampros/jargonary">
        <GithubLogoIcon size={20} aria-hidden />
        GitHub
      </a>
    </nav>
  )
}

export function SiteHeader({
  siteName,
  base = '',
  operator = false,
}: {
  siteName: string
  base?: string
  operator?: boolean
}) {
  return (
    <header className="masthead">
      <div className="masthead-identity">
        <a className="wordmark" href={`${base}/`}>
          {siteName || 'Jargonary'}
        </a>
        <span className="masthead-label">{operator ? 'Operator console' : 'Shared glossary'}</span>
      </div>
      <div className="masthead-actions">
        {!operator && (
          <a
            className="editor-launch-link"
            href={`${base}/editor`}
            title="Create or edit a JSON or YAML file on your computer"
          >
            <NotePencilIcon size={20} weight="duotone" aria-hidden />
            Glossary Editor
          </a>
        )}
        {operator && <span className="read-only">Read-only</span>}
        <ProjectLinks label="Project resources" />
      </div>
    </header>
  )
}

export function SiteFooter() {
  return (
    <footer className="site-footer">
      <div className="footer-about">
        <p className="footer-credit">Powered by Jargonary</p>
        <p>Your teams’ vocabulary, in context.</p>
      </div>
      <ProjectLinks label="Footer resources" />
    </footer>
  )
}
