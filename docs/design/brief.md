# Editorial dictionary UI

The user selected the first displayed design on November 3, 2026, with the requirement that the left alphabet navigation have a mobile-first responsive variant. This is a production UI inside the existing Go app, not a separate hosted prototype. The application name remains undecided; `site_name` controls the organization masthead.

## Visual references

- [Selected desktop reference](references/editorial-desktop.png): 1487 × 1058 pixels, composed for a 1440 × 1024 desktop viewport.
- [Mobile adaptation](references/editorial-mobile.png): 852 × 1846 pixels, composed for a 390 × 844 mobile viewport.

Both were generated with the built-in image-generation tool. The desktop direction requested a calm editorial dictionary: warm ivory, forest-green accents, serif term/context headings, readable sans-serif controls, numbered meanings, source attribution, alias text, a copy-link action, and a left alphabet rail. The mobile adaptation used the selected desktop image as its reference and requested a horizontal alphabet strip with 44px targets, a single-column reading layout, and no page-level horizontal overflow. These are design references, not raster assets substituted for working UI.

## Implementation

- Source Serif 4 (600/700) for editorial headings; DM Sans for controls and reading text. Both are bundled locally through Fontsource.
- Phosphor icons for search, link, caret, copy confirmation, and refresh. No bespoke icon drawings or decorative image assets are needed.
- Paper `#fcf9f3`, ink `#171a16`, green `#164c29`, muted text `#6f7469`, and hairline separators.
- At widths below 1000px the alphabet is a horizontal sticky strip. It scrolls independently, reveals the selected letter on load/resize, and supports arrow keys, Home/End, and Enter. The page remains constrained to its viewport.
- At 1000px and above it becomes a left rail. Coarse-pointer devices retain 44px targets even in that layout.
- Search/source controls stack on small phones. Copy sits below the term heading on phones. On very narrow screens, sense bodies lose their indentation to preserve reading width.
- The browse view groups senses into their entry and preserves source identity. Identical words from different sources are distinct results, not silently combined definitions.
- Definition HTML and preview metadata are server-generated. React enhances the page without requiring a Node runtime server. Native GET navigation preserves URLs and browser history.
- The operator UI reuses the typography, colors, and dividers, with separate source/configuration/diagnostic views. It is read-only and exists exclusively on the optional operator listener.

## Intentional differences from the illustrative mock

The implementation uses live source names and relationships. A single real entry attributes its meanings to its actual source; it does not invent the mock's separate Operations/Engineering attributions or its missing Asset system relationship. The joined search control is retained across sizes. An All control and numeric/other-character buckets make alphabet browsing reversible and complete. Muted text is slightly darker for contrast. Empty, missing, deprecated, stale-source, and pagination states extend the illustrative frame.

Preview images, interactive editing, authentication, and MCP are outside this UI pass. API-reference documentation is separate from the dictionary and operator interfaces.
