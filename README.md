# nanoinf

`nanoinf` is the official NanoInfluencer CLI for seed-based influencer discovery.

It is designed for both human operators and AI agents:
- JSON results go to `stdout`
- progress and errors go to `stderr`
- search context is saved locally so you can keep exploring with `nanoinf next`

## Install

### Homebrew

Coming soon.

### Install script

```bash
curl -fsSL https://raw.githubusercontent.com/nanoinfluencer/nano-cli/main/install.sh | sh
```

### Manual download

Download the latest archive from GitHub Releases, extract it, and move `nanoinf` into your `PATH`.

### Build from source

```bash
git clone https://github.com/nanoinfluencer/nano-cli.git
cd nanoinfluencer-cli
go build -o nanoinf .
```

## Authentication

Generate your NanoInfluencer access token from:

- `https://www.nanoinfluencer.ai/account/`

Then save it locally:

```bash
nanoinf auth token set <token>
nanoinf auth status
nanoinf whoami
```

## Quick Start

Resolve a creator URL:

```bash
nanoinf https://www.youtube.com/@theAIsearch
```

Find similar creators:

```bash
nanoinf similar https://www.youtube.com/@theAIsearch
nanoinf next
```

Enrich contact data:

```bash
nanoinf contact get --platform ytb --id UCIgnGlGkVRhd4qNFcEwLL4A
nanoinf contact fill --limit 20
```

Save creators:

```bash
nanoinf favorite add --platform ytb --id UCIgnGlGkVRhd4qNFcEwLL4A --project 12
nanoinf favorite fill --project 12 --limit 20
nanoinf hide add --platform ytb --id UCIgnGlGkVRhd4qNFcEwLL4A --project 12
```

## Search Filters

`nanoinf similar` supports these filters:

- `--has-email`
- `--country US` (repeatable)
- `--exclude-country JP` (repeatable)
- `--active-within 30`
- `--subs 10000:200000`
- `--views 1000:50000`
- `--posts 10:500`
- `--er 2:20`
- `--vr 5:50`

Example:

```bash
nanoinf similar https://www.youtube.com/@theAIsearch \
  --has-email \
  --country US \
  --country GB \
  --active-within 30 \
  --subs 10000:200000
```

## Output Contract

- Structured results are printed to `stdout` as JSON.
- Progress is printed to `stderr` when a TTY is attached.
- This makes `nanoinf` safe to use with `jq`, `tee`, and shell pipelines.

Example:

```bash
nanoinf similar https://www.youtube.com/@theAIsearch --has-email \
  | jq '.channels | length'
```

## Supported Platforms

- macOS: `amd64`, `arm64`
- Linux: `amd64`, `arm64`
- Windows: `amd64`

## Development

```bash
make test
make build
make release-snapshot
```

## Versioning

Tag releases with semantic versions such as `v0.1.0`.

## License

MIT
