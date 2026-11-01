# Design System: myjournal

This is the Stitch prompting reference for myjournal's existing interface. Product behavior in [requirements](docs/development/requirements.md) and the user's requested changes take precedence over generic design-skill defaults. Use the actual shared styles and templates when implementing designs.

## 1. Visual Theme & Atmosphere

Quiet, balanced daily writing software: density 4/10, variance 2/10, motion 1/10. Favor predictable symmetry, readable text and visible controls. Keep the established feature-card structure and functional navigation. Do not introduce an unrelated landing-page aesthetic.

## 2. Color Palette & Roles

| Role | Light | Dark |
| --- | --- | --- |
| Page canvas | #FFFFFF | #121212 |
| Raised surface | #F5F5F5 | #1E1E1E |
| Stronger surface | #EDEDED | #262626 |
| Primary ink | #1A1A1A | #EBEBEB |
| Secondary ink | #5F5F5F | #A8A8A8 |
| Decorative divider | #DCDCDC | #2E2E2E |
| Identifying control boundary | #767676 | #8A8A8A |

Use existing semantic success/error tokens only for truthful feedback, with accompanying text. No decorative accent palette, gradients or neon glows. The official full-color Google sign-in mark is a brand exception: retain its colors and aspect ratio, directly on the theme-specific button fill as in Google’s supplied light/dark examples. It is bundled locally; never substitute a monochrome CSS mask or fetch it from a third party at runtime.

## 3. Typography Rules

Retain the existing Segoe UI/system/-apple-system/BlinkMacSystemFont/Helvetica Neue/sans-serif stack. No remote font or serif UI. The Google sign-in button alone uses locally bundled Google Sans Medium as required by its branding guidance. Use weight, spacing and restrained clamp-based heading sizes for hierarchy. Body text has relaxed leading; informational prose can use a 65-character reading measure inside its container. The editable journal intentionally fills the available writing width rather than imposing that prose limit. Support formal English and Bahasa Baku, long labels and arbitrary plain-text user content.

## 4. Component Stylings

- Buttons use the existing pill/circle shapes, at least 44px height, visible focus and a light ink-colored border on hover. Keep disabled states semantic and theme-aware.
- The Google action keeps its localized Continue with Google label. Use the official light/dark fill, ink and border colors, Google Sans Medium at 14px/20px, 12px horizontal padding and 10px icon gap. Its decorative icon has an empty alt, 18px artwork without an added backing circle; the text supplies the accessible name.
- Cards retain subtle borders, established rounded corners and little visual elevation. Same-level landing cards share outer edges and gutters.
- The writing field uses the strong boundary token. Trash sits at the bottom right in a borderless normal-flow action area below the scrolling textarea. Reserve space below the text, not to its right, so the full writing width remains available and text is never obscured.
- Inputs and language menus retain visible labels, below-field options, keyboard dismissal and explicit Settings save. Error/recovery messages stay inline and readable.
- Loading, blank and failed states remain distinct. Never imply a saved entry or successful browser backup before acknowledgement.

## 5. Layout Principles

The shared container is capped at 68rem (1088px), including 1rem (16px) side gutters. On wide screens, the editor and its recovery banner fill that container's content width, aligning with the header's content edges. Keep narrower standalone informational reading pages intentional.

At widths up to 640px, the header has separate rows: left-aligned brand, right-aligned action controls and centered page navigation. Between 641px and 800px, retain the compact brand/actions row with centered page navigation below. Wider layouts retain centered application navigation between brand and actions.

No horizontal overflow or overlapping text. Retain the functional seven-column Monday-first calendar on mobile; do not apply a generic single-column rule to its date grid. Menus may float over the page when open, but must not hide their own trigger or extend off-screen. Check 320px, 390px, 640px and desktop layouts in both themes and languages.

## 6. Motion & Interaction

Keep the existing short control transitions and reduced-motion support. No perpetual pulses, shimmers, typewriters, spring dependencies or delayed list reveals: writing and recovery need stable, immediately usable controls. Theme remains in the header. Preserve all runtime data hooks, draft safeguards, account isolation, conflict resolution, confirmation and authentication recovery.

## 7. Anti-Patterns

No remote fonts, trackers, CDN assets, decorative emoji, neon gradients, fake metrics or marketing filler. Do not replace actual content with sample writing, change the plain-text editor into rich text, force asymmetric layouts, remove useful Help/Drive-access links, or copy static Stitch fixture logic into production. Existing typography, card structure, restrained motion and functional grids override the generic skill defaults.
