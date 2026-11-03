# UI checkpoint review

Status: checkpoint requested by the user before the full design-QA handoff was completed.

## Evidence

- Selected source: [editorial desktop](docs/design/references/editorial-desktop.png), 1487 × 1058, composed for 1440 × 1024 CSS pixels.
- Mobile source: [editorial mobile](docs/design/references/editorial-mobile.png), 852 × 1846, composed for 390 × 844 CSS pixels.
- Final browser captures: [desktop](docs/design/evidence/desktop-final.jpg), [phone](docs/design/evidence/mobile-final.jpg), and [tablet](docs/design/evidence/tablet-final.jpg).
- Combined comparison inputs: [desktop](docs/design/evidence/desktop-comparison-final.png) and [phone](docs/design/evidence/mobile-comparison-final.png). References are proportionally normalized into the corresponding 1440 × 1024 and 390 × 844 comparison panels. Browser device scale factor: 1.
- Earlier comparison: [initial mobile](docs/design/evidence/mobile-comparison-initial.png).

## Reviewed behavior and fixes

The implemented desktop rail and mobile strip were inspected in a browser. At 320, 390, and 768 CSS pixels there was no document-level horizontal overflow. Mobile letter targets measured 44 × 44 pixels. Selecting Z and resizing kept the active letter visible. Arrow navigation, search, source filtering, empty-letter states, native back navigation, relationship links, deprecation guidance, copy confirmation, and the search shortcut were exercised.

The first mobile comparison found the copy action still beside the term and a smaller, heavier title than the chosen reference. The action now sits below the heading on phones, the title scale/weight was adjusted, and spacing was tightened. Muted text contrast was improved, numeric labels were prevented from wrapping, and scrollbar-driven strip height changes were removed. The final combined images above were opened and reviewed after those changes.

The initial stylesheet was blocked by the server's CSP; self-hosted styles are now allowed and inline styles remain disallowed. The in-app browser passed interaction checks but failed several larger screenshot requests. Chrome was used for the final desktop/tablet/phone captures; no warnings/errors were reported in the inspected public-page console.

## Fidelity surfaces

- Typography: bundled Source Serif 4 and DM Sans preserve the editorial hierarchy; the final serif weight is lighter than the first implementation.
- Spacing/layout: flat reading surface, hairline dividers, desktop rail, and mobile strip follow the selected direction. The mobile copy action and typography were corrected after comparison.
- Colors: warm paper and forest green are retained; muted text is deliberately darker for readability.
- Assets/icons: no illustrative raster assets are required; standard Phosphor icons and locally bundled fonts are used. No screenshot is substituted for functional UI.
- Content: real meanings, aliases, and source attribution replace the illustrative mock's invented attributions/related term. All/numeric/other-character navigation and edge states are intentional additions.

## Remaining verification

The user accepted the current UI for a checkpoint. A formal final design-QA pass, focused-region captures, and the operator UI's complete interactive review were not finished before that checkpoint request. This document records evidence without claiming a completed visual QA gate. Backend route isolation and public-data redaction have automated Go coverage. Docker image execution still needs a running daemon.

final result: blocked
