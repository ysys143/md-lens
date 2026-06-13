---
name: release
description: Release a new version of md-lens (mdl) to GitHub — runs the quality gate, syncs the version constant, cross-compiles Linux/Windows binaries with checksums, tags, publishes a GitHub release, and verifies the download URLs. Use whenever the user asks to release, publish, tag, or ship a version (e.g. "release v1.2.0", "v1.0.2로 릴리즈해줘", "새 버전 배포", "publish the binaries"), even if they don't say the word "release".
---

# Release MD Lens

Publish version `vX.Y.Z` of MD Lens (binary: `mdl`) to https://github.com/benelog/md-lens/releases.
Takes the target version as an argument (e.g. `/release v1.0.2`); if omitted, bump the
patch number from the latest git tag.

Each step gates the next — a release with failing tests or a version mismatch is worse
than no release, so stop and report rather than pushing through a failure.

## 1. Quality gate

All four must pass before anything is tagged:

```bash
gofmt -l .               # must print nothing
go vet ./...
golangci-lint run ./...
go test ./...
```

## 2. Sync the version constant

The binary reports its own version (`--version`, help header), so `const version` in
`main.go` must equal the tag without the `v` prefix — a `v1.0.2` tag shipping a binary
that says `1.0.1` breaks the download-verification step and confuses bug reports.

```bash
grep 'const version' main.go     # must show X.Y.Z for tag vX.Y.Z
```

If it differs, edit it, rerun `go test ./...`, and verify:

```bash
go build -o mdl . && ./mdl --version   # expect: mdl X.Y.Z
```

## 3. Commit and push

Commit pending changes with a simple descriptive message (no conventional-commit
prefixes) and push to main. The tag must point at a pushed commit so the release's
source archive matches the binaries.

```bash
git add -A && git commit -m "<describe the change>" && git push origin main
```

## 4. Cross-compile the release binaries

Static, stripped builds (`-s -w` drops debug info, `-trimpath` removes local paths) —
single-file downloads with no runtime dependencies:

```bash
mkdir -p dist
GOOS=linux  GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/mdl-linux-amd64 .
GOOS=linux  GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/mdl-linux-arm64 .
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/mdl-windows-amd64.exe .
(cd dist && sha256sum mdl-* > SHA256SUMS.txt)
./dist/mdl-linux-amd64 --version    # host smoke test: expect mdl X.Y.Z
```

## 5. Tag and publish

Write release notes from the actual changes (`git log <prev-tag>..HEAD --oneline`),
not boilerplate. Keep the install table — it is what most visitors come for.

```bash
git tag vX.Y.Z && git push origin vX.Y.Z
gh release create vX.Y.Z \
  dist/mdl-linux-amd64 dist/mdl-linux-arm64 dist/mdl-windows-amd64.exe dist/SHA256SUMS.txt \
  --title "MD Lens vX.Y.Z" \
  --notes "<highlights + install table>"
```

Release-notes skeleton:

```markdown
<one-line summary of what changed>

## Changes
- <change 1>
- <change 2>

## Install
| Platform | Asset |
|---|---|
| Linux x86_64 | `mdl-linux-amd64` |
| Linux arm64 | `mdl-linux-arm64` |
| Windows x86_64 | `mdl-windows-amd64.exe` |

Single static binary, no dependencies. Checksums in `SHA256SUMS.txt`.
```

## 6. Verify the published release

The README's install commands use `releases/latest/download/...` — prove that path
works end-to-end before declaring success, and show the output:

```bash
gh release view vX.Y.Z --json assets --jq '.assets[] | "\(.name)  \(.size) bytes"'
cd /tmp && curl -sL -o mdl-verify https://github.com/benelog/md-lens/releases/latest/download/mdl-linux-amd64
chmod +x mdl-verify && ./mdl-verify --version && rm -f mdl-verify   # expect: mdl X.Y.Z
curl -sIL https://github.com/benelog/md-lens/releases/latest/download/mdl-windows-amd64.exe | grep -i '^HTTP' | tail -1   # expect 200
```

If verification fails, fix and re-upload assets with
`gh release upload vX.Y.Z <files> --clobber` — do not leave a broken release published.
