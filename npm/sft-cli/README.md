# sft-cli

Behavioral spec tool for UI screens, regions, events, state machines, and flows. SQLite-backed, single binary.

## Install

```bash
npm i -g sft-cli
# or
npx sft-cli show
```

## What it does

sft makes implicit UI structure explicit. Describe screens, regions, events, state machines, and flows in YAML — sft stores them in SQLite and gives you a typed, queryable spec.

```bash
sft init spec.yaml       # bootstrap from YAML
sft show                  # full spec tree with @refs
sft validate              # check for issues
sft export spec.yaml      # serialize DB state back to YAML
sft query screens         # list screens
sft query attachments     # attachments with content tracking
sft view                  # open in browser when the frontend bundle is embedded
```

If `sft view` reports that the frontend is not bundled, use a build that includes embedded web assets.

## Platforms

Prebuilt binaries for:
- Linux x64 / arm64
- macOS x64 / arm64 (Intel + Apple Silicon)
- Windows x64

Set `SFT_BINARY_PATH` to use a custom binary location.

## Documentation

See the full README at [github.com/lagz0ne/sft](https://github.com/lagz0ne/sft).

## License

MIT
