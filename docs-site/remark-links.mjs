import path from 'node:path'

// Resolve repository Markdown links without changing their canonical source files.
export function siteLink(url, base) {
  if (/^(?:[a-z][a-z\d+.-]*:|\/\/|#)/i.test(url)) return url
  const target = new URL(url, 'https://repo.invalid/docs/')
  const pathname = target.pathname
  const prefix = base.replace(/\/$/, '')
  const suffix = target.search + target.hash
  if (pathname.startsWith('/docs/') && pathname.endsWith('.md')) {
    const slug = pathname.slice('/docs/'.length, -3)
    return `${prefix}/${slug === 'index' ? '' : `${slug}/`}${suffix}`
  }
  if (pathname.startsWith('/docs-site/public/schema/')) {
    return `${prefix}${pathname.slice('/docs-site/public'.length)}${suffix}`
  }
  if (pathname.startsWith('/schema/')) return `${prefix}${pathname}${suffix}`
  const repoPath = pathname.slice(1)
  const kind = path.posix.extname(repoPath) ? 'blob' : 'tree'
  return `https://github.com/alampros/jargonary/${kind}/main/${repoPath}${suffix}`
}

export default function remarkLinks({ base }) {
  return (tree) => {
    function visit(node) {
      if (node.type === 'link' || node.type === 'definition' || node.type === 'image') {
        node.url = siteLink(node.url, base)
      }
      for (const child of node.children || []) visit(child)
    }
    visit(tree)
  }
}
