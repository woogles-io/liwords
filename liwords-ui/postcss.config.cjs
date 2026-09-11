// Mantine's PostCSS preset. Rsbuild discovers this automatically via
// postcss-load-config and runs it after @rsbuild/plugin-sass, so .scss files go
// Sass -> PostCSS and plain .css files go straight to PostCSS.
//
// The preset provides Mantine's rem()/em() helpers, its light-dark() shorthand,
// and mixins like `@mixin hover`. postcss-simple-vars supplies the breakpoint
// variables those mixins reference.
module.exports = {
  plugins: {
    "postcss-preset-mantine": {},
    "postcss-simple-vars": {
      variables: {
        "mantine-breakpoint-xs": "36em",
        "mantine-breakpoint-sm": "48em",
        "mantine-breakpoint-md": "62em",
        "mantine-breakpoint-lg": "75em",
        "mantine-breakpoint-xl": "88em",
      },
    },
  },
};
