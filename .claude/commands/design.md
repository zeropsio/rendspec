Render a visual design using the rendspec DSL.

When the user asks to create a UI mockup, architecture diagram, dashboard, landing page, or any visual design:

1. Write a `.rds` file using the rendspec YAML DSL (see CLAUDE.md for full reference)
2. Render it using: `rendspec render file.rds -o output.svg`
3. Show the user the rendered output using the Read tool on the SVG/PNG file

Key DSL patterns:
- Use `frame:` for containers, `text:` for labels
- Use `direction: row` for horizontal layouts, `column` (default) for vertical
- Use `flex: 1` for equal-width columns
- Use `layout: grid` + `columns: N` for grid layouts
- Use `fill: linear-gradient(...)` for gradients
- Use `components:` + `use:` for reusable elements
- Use `tokens:` + `$token.path` for design tokens
- Use `edges:` for connection lines between frames

$ARGUMENTS
