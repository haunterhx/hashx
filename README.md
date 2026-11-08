# hashx

**Concurrent file triage for defenders: SHA-256 IOC matching and Shannon entropy analysis in a single pass.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

`hashx` walks a directory tree, hashes every file, and flags two things:
 
- **IOC matches**: files whose SHA-256 is on a list of known-bad hashes you supply.
- **High-entropy files**: files whose bytes look random, a common trait of packed, compressed, or encrypted payloads.
It reads each file exactly once, feeding the hasher and the entropy histogram from the same buffer, and spreads the work across all CPU cores. It is a single static binary with no dependencies, and cross-compiles for Linux, Windows, and macOS.


```
$ hashx -iocs iocs.txt ./evidence
FLAG   ENTROPY  SIZE       SHA256                                                           PATH
IOC!   7.998    131072     43415fb4811247aebe8b617d3d43b717852c891570c8bd353bc61bede542d521 evidence/invoice.pdf.exe
IOC!   3.873    26         cb25e001078ec0ea8bb5dea3dbe31b713a8924fa9f512a15c3a1fa665a57458d evidence/dropper.sh
HIGH   7.999    262144     62accc177dfefc5ec675e3764a9af4f03827270d54571575c4921431d523da81 evidence/cache/blob.bin
 
scanned=1842 ioc_hits=2 high_entropy=1 errors=0 elapsed=212ms
```
 
---

## Contents
 
- [Installation](#installation)
  - [Option 1: Download a release binary](#option-1-download-a-release-binary)
  - [Option 2: Build from source with the build scripts](#option-2-build-from-source-with-the-build-scripts)
  - [Option 3: go install](#option-3-go-install)
- [Usage](#usage)
- [IOC file format](#ioc-file-format)
- [Exit codes](#exit-codes)
- [How it works](#how-it-works)
- [Limitations](#limitations)
- [Development](#development)
- [License](#license)

---

## Installation
 
### Option 1: Download a release binary
 
Pre-built binaries for Linux, Windows, and macOS (amd64 and arm64) are on the [Releases](https://github.com/haunterhx/hashx/releases) page.
 
**Linux / macOS**
 
```bash
VERSION=v1.0.0
OS=linux        # or: darwin
ARCH=amd64      # or: arm64
curl -LO "https://github.com/haunterhx/hashx/releases/download/${VERSION}/hashx_${VERSION}_${OS}_${ARCH}"
curl -LO "https://github.com/haunterhx/hashx/releases/download/${VERSION}/SHA256SUMS"
sha256sum -c SHA256SUMS --ignore-missing          # macOS: shasum -a 256 -c SHA256SUMS --ignore-missing
chmod +x "hashx_${VERSION}_${OS}_${ARCH}"
sudo mv "hashx_${VERSION}_${OS}_${ARCH}" /usr/local/bin/hashx
```
 
**Windows (PowerShell)**
 
```powershell
$Version = "v1.0.0"
Invoke-WebRequest "https://github.com/haunterhx/hashx/releases/download/$Version/hashx_${Version}_windows_amd64.exe" -OutFile hashx.exe
(Get-FileHash .\hashx.exe -Algorithm SHA256).Hash.ToLower()   # compare against SHA256SUMS
```
 
> **Note on antivirus:** the binaries are not code-signed. Statically linked Go binaries are sometimes flagged by heuristic AV engines. Verify the checksum against `SHA256SUMS`, or build from source (below).
 
### Option 2: Build from source with the build scripts
 
Requires **Go 1.22+** ([install Go](https://go.dev/dl/)) and **git**.
 
```bash
git clone https://github.com/haunterhx/hashx.git
cd hashx
```
 
**Linux / macOS / WSL / Git Bash** use `build.sh`:
 
```bash
./build.sh              # build for this machine        -> ./bin/hashx
./build.sh install      # build and install to ~/.local/bin (override: PREFIX=/usr/local/bin)
./build.sh all          # cross-compile all platforms   -> ./dist/ + SHA256SUMS
./build.sh test         # go vet + race-enabled tests
./build.sh clean        # remove ./bin and ./dist
```
 
**Windows** uses `build.ps1`:
 
```powershell
.\build.ps1             # build for this machine        -> .\bin\hashx.exe
.\build.ps1 install     # install to %LOCALAPPDATA%\Programs\hashx and add it to your user PATH
.\build.ps1 all         # cross-compile all platforms   -> .\dist\ + SHA256SUMS
.\build.ps1 test        # go vet + tests
.\build.ps1 clean       # remove .\bin and .\dist
```
 
If PowerShell blocks the script, run it once with:
 
```powershell
powershell -ExecutionPolicy Bypass -File .\build.ps1 install
```
 
After `install`, open a **new** terminal and confirm:
 
```bash
hashx -version
# hashx v1.0.0 (commit c2ceeba, built 2026-11-08T12:41:40Z, linux/amd64)
```
 
What the scripts do:
 
| Step | Detail |
|---|---|
| Check Go | Fails with install instructions if Go is missing or older than 1.22 |
| Stamp version | Takes the version from the latest git tag (`git describe`), plus the commit hash and the build date |
| Build | `CGO_ENABLED=0`, `-trimpath`, `-ldflags "-s -w"` produce a static, stripped binary with no local paths embedded |
| Cross-compile (`all`) | linux, windows, and darwin on amd64 and arm64, plus a `SHA256SUMS` manifest |
| Install | Copies the binary into a user-writable directory; no `sudo` or admin needed |
 
Override the version with `VERSION=1.2.3 ./build.sh all` or `.\build.ps1 all -Version 1.2.3`.

## Usage
 
```
hashx [flags] [path]
```
 
The path can be passed as the last argument or with `-path`. It defaults to the current directory.
**Flags must come before the path**: `hashx -json ./dir` works, `hashx ./dir -json` does not.
 
| Flag | Default | Description |
|---|---|---|
| `-path` | `.` | Directory to scan (alternative to the positional argument) |
| `-iocs` | *(none)* | File of SHA-256 IOCs, one per line |
| `-entropy` | `7.2` | Flag files at or above this entropy (bits per byte, 0–8) |
| `-exclude` | `.git,node_modules,.venv,__pycache__` | Comma-separated directory names to skip (`""` disables) |
| `-maxmb` | `512` | Skip files larger than this many MB (`0` = no limit) |
| `-workers` | number of CPUs | Concurrent hashing workers |
| `-json` | off | Emit one JSON object per file (JSON Lines) instead of the table |
| `-version` | | Print version, commit, and build info |
 
### Examples
 
```bash
# Entropy-only triage of a downloads folder
hashx ~/Downloads
 
# Hunt for known-bad files on a mounted image
hashx -iocs case-1234-iocs.txt /mnt/evidence
 
# Stricter threshold, larger files, include .git
hashx -entropy 7.8 -maxmb 4096 -exclude "" /srv
 
# JSON Lines for jq, a SIEM, or a spreadsheet
hashx -json -iocs iocs.txt /mnt/evidence > triage.jsonl
jq -r 'select(.ioc_hit) | .path' triage.jsonl
 
# Use in a script: act on IOC hits by exit code
hashx -iocs iocs.txt /data > report.txt
rc=$?
if   [ "$rc" -eq 2 ]; then echo "IOC match found, see report.txt"
elif [ "$rc" -ne 0 ]; then echo "scan failed (exit $rc)" >&2
fi
```
 
### Output
 
**Table mode** (the default) lists only flagged files, sorted by entropy. The summary line is written to **stderr**, so stdout stays clean for redirection.
 
| Flag | Meaning |
|---|---|
| `IOC!` | SHA-256 matches the IOC list. Shown in red in a terminal |
| `HIGH` | Entropy is at or above the threshold |
 
When stdout is not a terminal, colour is turned off automatically. Set `NO_COLOR=1` to force it off.
 
**JSON mode** (`-json`) emits every scanned file, flagged or not:
 
```json
{"path":"evidence/dropper.sh","size":26,"sha256":"cb25e001...","entropy":3.873,"ioc_hit":true,"high_entropy":false}
```
 
---
 
## IOC file format
 
Plain text, one SHA-256 per line:
 
```
# Lines starting with # are comments; blank lines are ignored
cb25e001078ec0ea8bb5dea3dbe31b713a8924fa9f512a15c3a1fa665a57458d
61DE98F2EE05A42D0963CEA8514333330A37E117C2AE17529C7E6E12691F1359
```
 
- Hashes are case-insensitive, and surrounding whitespace is trimmed.
- Windows (CRLF) line endings are handled.
- Lines that are not exactly 64 characters are ignored.
- A missing IOC file is a hard error (exit `1`). It never silently degrades to "no IOCs".
Build a list from files you already have:
 
```bash
find ./known-bad -type f -exec sha256sum {} + | awk '{print $1}' > iocs.txt
```
 
---
 
## Exit codes
 
| Code | Meaning |
|---|---|
| `0` | Scan completed, no IOC matches |
| `1` | Error: bad arguments, missing path, or unreadable IOC file |
| `2` | Scan completed and **at least one IOC matched** |
 
High-entropy files alone do not change the exit code.
 
---
 
## How it works
 
```
 walker ──paths──► worker 1 ─┐
 (WalkDir)        worker 2 ─┼──results──► collector ──► table / JSON
                  worker N ─┘
```
 
1. A walker goroutine traverses the tree with `filepath.WalkDir`. It sends only regular files, does not follow symlinks, and skips excluded directories.
2. N workers each open a file and stream it through a 64 KB buffer into a SHA-256 hasher and a 256-bucket byte histogram at the same time.
3. Shannon entropy is computed from the histogram:
   **H = − Σ p(b) · log₂ p(b)**, over all byte values *b*, giving a value between 0 and 8 bits per byte.
4. The hash is looked up in the IOC set (an O(1) map lookup), and results are collected and printed.
| Typical content | Entropy |
|---|---|
| Plain text, source code | 4.0 – 5.5 |
| Native executables (unpacked) | 5.0 – 6.5 |
| Compressed archives (`.zip`, `.gz`, `.deb`) | 7.5 – 8.0 |
| Encrypted data, packed malware | 7.9 – 8.0 |
 
---
 
## Limitations
 
Know what this tool does **not** do before relying on it:
 
- **Exact-match hashing only.** Changing a single byte produces a different SHA-256. Use fuzzy hashing (ssdeep, TLSH) or YARA to catch variants.
- **Whole-file entropy.** A small encrypted section inside a large ordinary file is averaged away. Sliding-window entropy would catch it.
- **High entropy is not malicious.** Archives, media, installers, and `.git` objects all score high. Treat `HIGH` as "look closer", not as a verdict.
- **Entropy is not randomness.** A file containing every byte value equally often scores exactly 8.0 even when perfectly ordered.
- **No content inspection.** File types, magic bytes, and PE headers are not parsed.
- **Point-in-time results.** Files modified during a scan may produce inconsistent results. Scan read-only mounts or images for forensic work.
---
 
## Development
 
```bash
go vet ./...
go test -race ./...
./build.sh test
```
 
Acceptance tests run against a deterministic corpus (identical bytes, and therefore identical hashes, on every machine):
 
```bash
go build -o bin/hashx ./cmd/hashx
./scripts/run_tests.sh ./bin/hashx
```
 
### Project layout
 
```
cmd/hashx/          CLI entry point: flags, output, exit codes
internal/scan/      walker, worker pool, per-file analysis, Result type
internal/entropy/   Shannon entropy
internal/ioc/       IOC list loading
scripts/            test-data generators and acceptance test runner
build.sh, build.ps1 build helpers for end users
```
 
### Releasing
 
```bash
git tag -a v1.1.0 -m "v1.1.0"
git push origin v1.1.0
./build.sh all
gh release create v1.1.0 dist/* --title "hashx v1.1.0" --generate-notes
```
 
---
 
## License
 
[MIT](LICENSE) © 2026 haunterhx
 