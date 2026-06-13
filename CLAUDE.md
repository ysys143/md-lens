# Markdown Lens (mdl) — developer guide

A terminal Markdown viewer written in Go. Module path: `github.com/benelog/md-lens`.
Requires Go 1.24+.

## Build & test

```bash
go build -o mdl .        # build the binary -> ./mdl
go test ./...            # run the unit tests
```

## Quality gate

Run all of these before considering any change done (a PostToolUse hook in
`.claude/settings.json` also runs them on every edited .go file):

```bash
gofmt -l .               # must print nothing
go vet ./...
golangci-lint run ./...
go test ./...
```

## Releasing

Use the `/release` skill (`.claude/skills/release/SKILL.md`) — it runs the quality gate,
syncs `const version` in `main.go` with the tag, cross-compiles Linux/Windows binaries
with checksums, tags, publishes the GitHub release, and verifies the download URLs.

## Project layout

```
main.go                 entry point (flags → detect terminal → render)
internal/ansi/          SGR escapes, OSC 8 links, visible-width measurement
internal/cli/           option parsing
internal/term/          terminal capability detection + --caps report
internal/highlight/     per-language regex tokenizers, theme, highlighter
internal/image/         image load/scale + kitty / iTerm2 / half-block emitters
internal/heading/       font-image headings + styled-text fallback
internal/render/        goldmark AST visitor, layout, wrapping
docs/                   demo.md + README screenshot
```

## Key invariants

- `ansi.Width` is the single source of truth for visible width (ANSI/OSC8 skipped,
  East-Asian wide chars = 2). Never use `len()` for layout.
- `ColorDepth None` must produce zero escape bytes — piped output stays byte-clean.
- All rendered output goes through the render `Context` (line/blank/pushPrefix);
  its prefix stack + pendingMarker drive indentation.
- Tokenizer rules are ordered: comments → strings → annotations → numbers → keywords →
  types/builtins → functions → operators → punctuation. Patterns are `\G`-anchored
  regexp2 (stdlib regexp cannot express the lookaheads).
- Inline styles close with their specific off-codes (BoldOff etc.), never Reset,
  so nested styles survive.

## Dependencies

| Concern | Package |
|---|---|
| Markdown parsing (CommonMark + GFM) | `github.com/yuin/goldmark` |
| Regex tokenizers (lookahead + `\G`) | `github.com/dlclark/regexp2` |
| Font rasterization + image scaling | `golang.org/x/image` (embedded Go bold font) |
| Terminal size / TTY detection | `golang.org/x/term` |

## Regenerating the README screenshot

```bash
go build -o mdl .
kitty --detach --config NONE -o linux_display_server=x11 -o font_size=9 \
  -o initial_window_width=96c -o initial_window_height=50c -o window_padding_width=8 \
  -T mdl-shot sh -c "$PWD/mdl $PWD/docs/demo.md; exec sleep 600"
sleep 5 && import -window mdl-shot docs/screenshot.png   # ImageMagick, X11/XWayland
```
