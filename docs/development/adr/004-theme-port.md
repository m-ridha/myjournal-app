# 004: Port Monochrome rather than run Hugo

The Monochrome Hugo theme will be ported into Go templates and embedded static assets because the app is a single binary and the guide explicitly forbids bringing Hugo into the runtime path. This keeps the theme easier to maintain and avoids runtime dependencies on a static site generator.
