# Pinable Desktop runtime distribution

## Scope

`Pinable desktop runtime` builds **CLIProxyAPI component runtimes**, not Desktop
installers. It adds a fork-owned workflow and standard-library Python packaging
tools under `scripts/pinable/`, without changing upstream server code, release
workflows, Go dependencies, model catalogs, or Desktop's Go build orchestrator.

The Desktop build contract supplied for this integration requires source revisions
and SHA256 values to remain locked, generated binaries/archives to stay out of Git,
Windows/macOS/Linux assets, fail-fast validation before Wails, and prebuilt
single-target assets through the absolute `CLIPROXYAPI_COMPONENT_ASSET` variable.

The supplied document does **not** define Desktop's `component-lock.json` schema
or the exact `CLIPROXYAPI_COMPONENT_ASSET_DIR` naming convention. The Desktop source
repository was not available through the connection used for this change.
Consequently, `runtime-index.json` is a **distribution manifest**, NOT a drop-in
replacement for `pinable-desktop/internal/assets/cliproxyapi/component-lock.json`.
No tool here reads, rewrites, invents, or relaxes that lock schema. End-to-end Wails
packaging is not claimed by this component pipeline.

## Targets and profile

| Target | Native CI runner | Archive |
| --- | --- | --- |
| `windows/amd64` | `windows-2025` | ZIP |
| `windows/arm64` | `windows-11-arm` | ZIP |
| `darwin/amd64` | `macos-15-intel` | tar.gz |
| `darwin/arm64` | `macos-15` | tar.gz |
| `linux/amd64` | `ubuntu-22.04` | tar.gz |
| `linux/arm64` | `ubuntu-22.04-arm` | tar.gz |

All outputs use `desktop-portable`: `CGO_ENABLED=0`, Go's baseline CPU settings,
`-trimpath`, and explicit version/full source SHA/source commit timestamp metadata.
This removes Linux dynamic-interpreter dependence and avoids requiring a compiler
or development SDK just to run the component.

**Dynamic library plugins are NOT supported by this profile.** It does not replace
upstream's plugin-enabled distribution. The limitation is recorded as
`capabilities.dynamic_library_plugins=false`. Preserve upstream's default
plugin-disabled configuration.

macOS executables are individually ad-hoc signed and verified before hashing.
These are thin amd64/arm64 binaries, not Universal 2. Desktop's application packager
can choose the matching component for each application slice. Developer ID
signing/notarization and final bundle signing remain the Desktop release operator's
responsibility. Windows executables are unsigned. No signing credentials are
required, added, printed or modified.

## Downloads and triggers

Workflow: `.github/workflows/pinable-runtime.yml` (Actions name:
**Pinable desktop runtime**).

- Every `main` push builds the current source, including a merged weekly sync.
- Packaging-related PRs generate native-tested candidate downloads.
- **Run workflow** can build a selected branch without publishing a release.
- Explicit manual `publish_release=true` on **main only** publishes a dedicated
  `pinable-runtime-<version>` prerelease after all six packages and quality gates
  pass. The default is **false**. It does not replace the normal latest release
  or deploy anything.

The source revision is captured once. Builds do not pull source, refresh remote
model catalogs, modify source files, or use a moving branch as their metadata.
The builder requires clean tracked/untracked source and exactly the Go version in
`go.mod`; dependency downloads retain normal Go checksum checks.

Individual `pinable-runtime-<os>-<arch>` Actions artifacts are retained for 30 days.
The complete **`pinable-runtime-all`** artifact is retained for 90 days. Links are
written into the workflow summary. GitHub login/access is needed for Actions
artifact downloads. For durable unauthenticated downloads from this public fork,
use the explicit prerelease path; GitHub supplies the release asset URLs.

Publication creates a draft first, uploads the complete verified set, then makes
it visible. Existing tags/assets are never clobbered. Interrupted publication leaves
a draft for operator inspection. The source must still equal `main` before publishing.
The default GitHub Actions token is used, not a personal token. Tags are created by
that job, not by an instruction to push tags manually. Do not push
`pinable-runtime-*` tags manually: upstream's independent release workflow has an
all-tags trigger. This new workflow deliberately has no tag trigger.

## Output contract

Each target contains an archive named
`CLIProxyAPI_pinable_<version>_<goos>_<goarch>.zip` on Windows or `.tar.gz` on
macOS/Linux, `<archive-name>.manifest.json`, and `smoke-report.json` in its individual
CI artifact. The complete artifact contains all six archives and manifests,
`runtime-index.json`, `SHA256SUMS`, and `validation.json` with all native reports.

Inside each archive there are exactly three regular files:

```text
cli-proxy-api[.exe]
LICENSE
runtime.json
```

No config, `.env`, credentials, auth directories, plugins, management web assets
or Desktop managed launcher are included. Paths, regular-file types and executable
modes are checked without extracting arbitrary paths. SHA256 is recorded separately
for the **archive** and **unpacked binary**. Desktop's binary lock must use the latter,
not the Actions ZIP digest.

Each manifest records schema/profile, target, source repository/full SHA/time,
component version, exact Go version, build flags, signing policy, host contract
version `1`, capabilities, license checksum, binary size/hash and archive size/hash.
`runtime.json` contains the same record without the recursive archive checksum.

Archive timestamps, member order, ownership and permissions are deterministic.
Build time is the **source commit timestamp**, not wall-clock packaging time.
Determinism for identical input binaries is tested; this does not claim all compiler
or signing hosts produce bit-identical binaries.

## Desktop integration

Choose the target and approve its source revision and **binary SHA256** against
Desktop's component lock. Obtain archive and manifest from the same validated run
or release. Do not trust an unrelated checksum merely because it matches a download.

From a checkout containing these tools (Python 3.10+):

```bash
python scripts/pinable/runtime.py verify \
  --manifest /absolute/downloads/<archive-name>.manifest.json \
  --expected-commit <approved-full-40-character-source-sha> \
  --expected-sha256 <approved-unpacked-binary-sha256>

python scripts/pinable/runtime.py stage \
  --manifest /absolute/downloads/<archive-name>.manifest.json \
  --expected-commit <approved-full-40-character-source-sha> \
  --expected-sha256 <approved-unpacked-binary-sha256> \
  --output /absolute/component-staging/cli-proxy-api
```

Use `cli-proxy-api.exe` on Windows. Staging requires a new absolute destination,
never replaces a running/existing runtime, and checks archive contents, CPU headers,
metadata and both hashes before writing the executable.

Then use the **existing** Desktop single-target override and validation:

```bash
export CLIPROXYAPI_COMPONENT_ASSET=/absolute/component-staging/cli-proxy-api
# In the Desktop repository, not CLIProxyAPI:
go run ./pinable-scripts/cmd/pinable-build doctor --package
go run ./pinable-scripts/cmd/pinable-build package --platform linux
```

PowerShell:

```powershell
$env:CLIPROXYAPI_COMPONENT_ASSET = 'C:\components\windows-amd64\cli-proxy-api.exe'
go run ./pinable-scripts/cmd/pinable-build doctor --package
go run ./pinable-scripts/cmd/pinable-build package --platform windows
```

Use `--platform macos` for macOS. CodeGraph remains independently required and is
not supplied here. An older revision, different build flags or binary hash in the
lock **must still fail**. Approving this runtime may require a reviewed Desktop
lock/tooling change in that repository. Rebuilding with its previous
`components --update-lock` recipe is not guaranteed to reproduce this profile's
hash. This pipeline never bypasses the mismatch.

## Validation gates

The full normal-profile Go suite and existing sync-script tests run on Linux.
Each of the six **native architectures** runs portable-profile host regressions,
builds the server, verifies the archive, extracts the exact binary and tests:

- both version flags without config or file creation;
- new/legacy discovery output parses as JSON;
- authenticated runtime metadata matches version, full SHA, timestamp and PID;
- public credentials cannot authorize management APIs;
- ephemeral key survives a proven config-key rotation without being persisted;
- unauthorized shutdown fails; authenticated loopback shutdown exits cleanly;
- monitored parent exit stops the runtime;
- macOS ad-hoc signature survives transport and extraction.

The six-target index is emitted only after quality and all native jobs pass.
Publication verifies the complete set, manifests, reports and checksums again.
These are local/loopback tests, **not live provider API/account tests**.

## Local build and tests

```bash
python scripts/pinable/test_runtime.py -v
python scripts/pinable/runtime.py build --target linux/amd64 --output /tmp/new-runtime-output
python scripts/pinable/smoke_runtime.py --artifact-dir /tmp/new-runtime-output
```

Builds use `go build ./cmd/server` and exactly the `go.mod` toolchain. Native macOS
is required for signing. Optional `--version` accepts only a safe label; otherwise
source timestamp plus SHA identifies the build. The tools need no pip modules,
zip/unzip, Node, Docker, GoReleaser or global environment changes. Generated assets
must not be committed.
