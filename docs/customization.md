---
title: Branding and themes
editUrl: https://github.com/alampros/jargonary/edit/main/docs/customization.md
---

Jargonary supports bundled themes, a company name, and ordered local CSS/JavaScript files without rebuilding the image. These settings apply to both the public dictionary and the optional operator UI. They resolve at startup; restart after changing settings or custom files.

## Quick start

```yaml
site_name: Example Corporation
theme: github
custom_css:
  - branding/colors.css
  - branding/layout.css
custom_js:
  - branding/hooks.js
```

Paths are relative to the configuration file. With no configuration file, environment-supplied relative paths resolve from the process working directory. Absolute paths also work. In Docker, mount your configuration and branding files read-only:

```sh
docker run --rm -p 8080:8080 \
  -v "$PWD/config:/config:ro" \
  ghcr.io/your-org/jargonary:latest -config /config/config.yaml
```

Replace the image name with your built/published image. Include your usual sources and database settings in the config; this example only illustrates the branding mount.

## Settings and environment variables

Environment variables override the corresponding YAML settings; YAML overrides built-in defaults. An environment list replaces the entire YAML list. The operator's resolved configuration records which settings came from the environment.

| YAML | Environment variable | Default |
| --- | --- | --- |
| `site_name` | `JARGONARY_SITE_NAME` | `Jargonary` |
| `theme` | `JARGONARY_THEME` | `default` |
| `custom_css` | `JARGONARY_CUSTOM_CSS` | `[]` |
| `custom_js` | `JARGONARY_CUSTOM_JS` | `[]` |

```sh
export JARGONARY_SITE_NAME='Example Corporation'
export JARGONARY_THEME=github-dark
export JARGONARY_CUSTOM_CSS='["/config/branding/colors.css", "/config/branding/layout.css"]'
export JARGONARY_CUSTOM_JS='["/config/branding/hooks.js"]'
go tool task run
```

List variables use JSON arrays, preserving spaces and commas in filenames. Use `[]` to clear a list. Unknown themes, malformed arrays, missing/unreadable files, and files with the wrong extension fail startup with an actionable error. CSS files must end in `.css`, JavaScript in `.js`; each file can be up to 4 MiB. They must be readable by the container's UID 65532.

`site_name` controls the header on both UIs and the site portion of definition titles/social previews. Dictionary source names belong to the source documents and remain independent.

## Included themes

| Name | Appearance |
| --- | --- |
| `default` | Original warm paper, forest green, serif headings |
| `github` | White/slate surfaces, blue accent, sans-serif headings |
| `github-dark` | Near-black/slate surfaces, light blue accent, sans-serif headings |
| `base64-dark` | Neutral charcoal, muted cyan, serif headings |
| `nord` | Blue-grey surfaces, pale cyan, serif headings |

These are Jargonary palettes; the GitHub/Nord-inspired palettes are not official branded themes. `base64-dark` is the included theme's name, not a file encoding. Theme choice is fixed by deployment configuration, including the native form-control light/dark scheme. It does not follow OS preferences or require local browser storage.

## CSS variables

All application colors use semantic `--jg-*` variables. Component styles live in the `jargonary` CSS cascade layer. Bundled theme variables and custom styles are unlayered, so they override the base even when Vite injects development CSS later. Prefer overriding variables over copying component selectors.

The order is **base styles → selected theme → custom CSS files in listed order**. Later custom files win for equal-specificity rules. No `!important` is required for ordinary overrides.

```css
/* branding/colors.css */
:root {
  --jg-accent: #6941c6;
  --jg-accent-hover: #53389e;
  --jg-on-accent: #ffffff;
  --jg-font-heading: var(--jg-font-body);
  --jg-radius: 4px;
}
```

| Variable | Used for |
| --- | --- |
| `--jg-bg` | Page background |
| `--jg-fg` | Primary text and form input text |
| `--jg-secondary` | Definition previews and supporting text |
| `--jg-muted` | Pronunciation, part of speech, metadata icons, placeholders |
| `--jg-accent` | Links, active alphabet/tab states, buttons, focus rings |
| `--jg-accent-hover` | Hovered links and buttons |
| `--jg-on-accent` | Text/focus indicator on accent-filled controls |
| `--jg-border` | Dividers and ordinary borders |
| `--jg-border-strong` | Header and form-control borders |
| `--jg-surface` | Configuration output, copy fallback fields |
| `--jg-hover` | Hovered controls and metadata icons |
| `--jg-warning` | Deprecated/stale status text |
| `--jg-warning-bg` | Deprecation notice background |
| `--jg-warning-border` | Deprecation notice border |
| `--jg-danger` | Failure status text |
| `--jg-shadow` | Tooltip shadow color |
| `--jg-font-body` | Body/UI font stack (default: bundled DM Sans) |
| `--jg-font-heading` | Heading font stack (default: bundled Source Serif 4) |
| `--jg-font-mono` | Configuration/code font stack |
| `--jg-font-size` | Root font size (default: `16px`) |
| `--jg-radius` | Primary search/filter corner radius (default: `8px`) |
| `--jg-page-padding` | Base page padding (default: `24px`; responsive rules may override it) |
| `--jg-content-width` | Desktop dictionary maximum width (default: `1728px`) |

Test contrast for text, focus rings, selected states, and warning/failure notices after changing colors. Check both UIs, narrow screens, and definition pages.

### Scope changes to one UI

The HTML element exposes `data-ui="public"` or `data-ui="admin"`, and `data-theme="<selected name>"`:

```css
html[data-ui="public"] {
  --jg-font-heading: Georgia, serif;
}

html[data-ui="admin"] {
  --jg-font-size: 15px;
}
```

Full CSS overrides can target existing classes such as `.masthead`, `.wordmark`, `.term-row`, `.definition`, and `.operator-panel`. These implementation selectors can evolve; the documented tokens and `data-ui`/`data-theme` attributes are the preferred customization interface.

## JavaScript hooks

Scripts are classic external scripts with `defer`, included in listed order. They are served from the same origin; inline script injection is unnecessary. React mounts asynchronously. Use the ready event for hooks that inspect the rendered UI, and check the readiness marker in case the script loaded after mounting:

```js
// branding/hooks.js
function onReady() {
  const mode = document.documentElement.dataset.jargonaryReady;
  if (mode === 'public') {
    document.documentElement.classList.add('company-dictionary');
  }
}

window.addEventListener('jargonary:ready', onReady);
if (document.documentElement.dataset.jargonaryReady) onReady();
```

The marker and event identify `public` or `admin`. Hooks should be idempotent because HMR can remount components. Prefer changing CSS variables, classes, or content outside React-owned nodes; React can replace direct edits inside its UI. Top-level ES module imports are not supported in these `.js` files.

## Asset delivery

Custom CSS and JavaScript files are served publicly and loaded on both UIs. JavaScript runs with the page’s privileges, including on the operator UI. This is potentially dangerous: use only files you trust, keep secrets out of them, and use at your own risk.
