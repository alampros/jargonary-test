---
title: Branding and themes
editUrl: https://github.com/alampros/jargonary/edit/main/docs/customization.md
---

Branding applies to the public glossary and operator UI. Restart after changing settings or custom files.

```yaml
site_name: Example Corporation
theme: github
custom_css:
  - branding/colors.css
  - branding/layout.css
custom_js:
  - branding/hooks.js
```

Paths resolve relative to the config file, or the working directory when no config is used. In containers, mount files read-only and make them readable by UID 65532.

## Settings

| YAML | Environment override | Default |
| --- | --- | --- |
| `site_name` | `JARGONARY_SITE_NAME` | `Jargonary` |
| `theme` | `JARGONARY_THEME` | `default` |
| `custom_css` | `JARGONARY_CUSTOM_CSS` | `[]` |
| `custom_js` | `JARGONARY_CUSTOM_JS` | `[]` |

Environment lists are JSON arrays and replace YAML lists; `[]` clears them. CSS/JS extensions must match, and each file is limited to 4 MiB. Invalid themes, arrays, or files fail startup.

`site_name` changes UI headers and page titles, not source names.

## Themes

Choose `default` (warm paper/green), `github` (light/blue), `github-dark` (dark/blue), `base64-dark` (charcoal/cyan), or `nord` (blue-grey/cyan). Theme selection is fixed by configuration, not OS preferences.

## CSS

Override semantic `--jg-*` variables rather than component selectors. Load order is base → theme → custom files in listed order. Component styles use the `jargonary` cascade layer; unlayered overrides also take precedence over Vite-injected styles.

```css
:root {
  --jg-accent: #6941c6;
  --jg-accent-hover: #53389e;
  --jg-on-accent: #ffffff;
  --jg-font-heading: var(--jg-font-body);
  --jg-radius: 4px;
}
```

See [theme palettes](../internal/themes/default.css) and [layout/font tokens](../ui/src/styles.css) for the full variable set. Scope overrides with `html[data-ui="public"]`, `html[data-ui="admin"]`, or `html[data-theme="github"]`. Component class names can change; tokens and these attributes are the customization interface.

## JavaScript hooks

Files load as classic deferred scripts in listed order; top-level module imports are unsupported. React mounts asynchronously:

```js
function onReady() {
  const mode = document.documentElement.dataset.jargonaryReady;
  if (mode === 'public') {
    document.documentElement.classList.add('company-glossary');
  }
}

window.addEventListener('jargonary:ready', onReady);
if (document.documentElement.dataset.jargonaryReady) onReady();
```

The readiness marker is `public` or `admin`. Keep hooks idempotent for HMR remounts. React may overwrite direct edits to its nodes; prefer CSS variables, classes, or content outside its tree.

Custom assets are public and load on both UIs. Scripts run with page privileges, including operator access; use trusted code and keep secrets out of assets.
