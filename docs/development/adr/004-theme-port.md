# 004: Port Monochrome rather than run Hugo

The Monochrome Hugo theme will be ported into Go templates and embedded static assets because the app is a single binary and the guide explicitly forbids bringing Hugo into the runtime path. This keeps the theme easier to maintain and avoids runtime dependencies on a static site generator.

## Implementation review (2026-10-10)

The wording above described the original plan. The actual implementation is a custom Go-template/CSS interface inspired by Monochrome, not an identified port of its source files. Hugo remains absent from the runtime and build.

The user identified the [Hugo Monochrome listing](https://themes.gohugo.io/themes/hugo-theme-monochrome/) as the visual reference and recalled no copying. Compared the 18 current local CSS/HTML/JavaScript/SVG files with 122 source files in [kaiiiz/hugo-theme-monochrome](https://github.com/kaiiiz/hugo-theme-monochrome) at commit 98923c5fda5a4a6755adc6f03721f6d8db4dd060. Found no identical files, no matching sequences of five substantive lines after removing whitespace, and no exact matches for the local SVG geometry strings. The upstream source has no `mj-*` component vocabulary. Local history starts with app-specific `--mj-*` tokens and Go templates in c665672; later commits evolve those files rather than importing an upstream theme tree.

Upstream's LICENSE identifies MIT, copyright 2021 kaiiiz. No upstream theme source version is bundled or has been identified as copied; the commit above is the comparison snapshot, not a vendored version. Record Monochrome as visual inspiration and retain existing project licensing. This comparison is evidence of no detected source copying, not proof covering heavily rewritten snippets or every historical upstream revision.

The four local SVGs have separate app history. Sun/moon began inline in the app, gear geometry was changed for evenly spaced teeth, and trash came from the user-provided SVG example. None matched the inspected theme geometry. The trash example's original source/license was not supplied; this review does not establish provenance for that separate third-party snippet. The favicon was added separately in 9f5db7d and was not part of the text-source comparison.
