# Pinable upstream synchronization

## Baseline and invariant

The 2026-09-20 synchronization integrates `router-for-me/CLIProxyAPI` at
`61fdfc341b96178a8dcb53f2efc46cbc341d267c` with the fork at
`491e5413d084f74614f10834d63b76d288fb3e56`. Their shared ancestor is
`44e62bc8acc2f224bff9c62d222717d3f6723dea`: 153 upstream commits and four
fork-only commits were present before integration.

Preserve both histories. Use a normal merge commit for synchronization PRs,
**not squash or rebase**, and do not reset the fork to upstream. Keeping the
upstream commit as an ancestor lets Git recognize already-integrated changes
on the next sync. Never resolve all conflicts with blanket `ours` or `theirs`.

## Host contract that must survive a sync

- `--version` / `-version` must work without loading configuration or starting
  services. Discovery JSON output must not contain the normal startup banner.
- `--parent-pid` retains parent-process monitoring and graceful cancellation.
- The authenticated `/v0/management/runtime-info` and
  `/v0/management/runtime-shutdown` endpoints retain contract version `1`.
  Shutdown additionally checks the actual loopback peer, not a forwarded header.
- `CLIPROXY_EPHEMERAL_API_KEY` / `Builder.WithEphemeralAPIKey` remain in-memory,
  service-scoped credentials. They must not enter persisted API-key configuration
  or appear in the authentication principal, and must survive plugin/config reload.
- Upstream startup paths and SDK initialization remain intact, including discovery,
  provider login modes, plugin hooks, and `discoveryManager` initialization.

## Conflict reduction implemented in this sync

The three actual conflicted files were `cmd/server/main.go`,
`cmd/server/main_test.go`, and `sdk/cliproxy/builder.go`.

Host version detection, environment handling, and safe-mode adaptation live in
`cmd/server/host_runtime.go`, with dedicated tests in `host_runtime_test.go`.
The upstream safe-mode function and `main_test.go` are retained unchanged from
the pinned upstream revision. This avoids repeatedly competing for the top of
upstream's test file or extending upstream function signatures.

The ephemeral-key builder option and initialization live in
`sdk/cliproxy/builder_runtime.go`. The upstream service constructor literal is
retained, followed by one host-specific initialization call. This prevents a
long fork-only field name from reformatting the entire literal and colliding
with newly added upstream fields. Runtime fields in the management handler
are separated into their own formatting group for the same reason.

Regression tests cover clean discovery output, upstream safe-mode decisions,
runtime-key isolation, unchanged configured keys, repeated reloads without
provider duplication, and preservation of upstream discovery initialization.

The upstream-only PR retargeting and AGENTS modification policies are scoped to
`router-for-me/CLIProxyAPI`. They must not redirect or close normal fork sync PRs.
The fork's own validation workflow has read-only repository permissions.

## Bootstrap test stability

The initial full Go run exposed an intermittent failure in the upstream
`TestCodexWebsocketsExecutor_BootstrapBuffering_StatusBearingErrorAfterTimeoutDeliveredInStream`.
The mock server could advance time before the executor captured its bootstrap
start. Related HTTP fixtures also signaled before reading the starting clock.
A shared test-only handshake now captures the start before allowing the server
to advance time. Production executor behavior and test assertions are unchanged.
The five affected timeout tests passed 100 repetitions each during integration;
the persistent CI repeats them 20 times in addition to the full suite.

## Next synchronization

Use Bash, Git, Python 3 for the script tests, and the Go version declared in
`go.mod`. Start on the desired current fork branch with a clean working tree.

```bash
git switch main
git pull --ff-only origin main
bash scripts/pinable/sync-upstream.sh
```

The script adds the official `upstream` remote only when it is missing; an
existing remote is displayed and used without silently rewriting its URL.
A branch, tag, or full commit SHA can replace the default `main` argument.
The target is fetched once and pinned to its resolved commit for this run.

It creates a dedicated `sync/upstream-<date>-<sha>` branch, starts a real merge,
builds the server, and runs `go test ./...` before creating a merge commit.
It never pushes a branch, force-updates a ref, discards changes, or overwrites
an existing sync branch. A repeated sync to an already-integrated commit is a no-op.

When conflicts occur, inspect both implementations, retain the host contract,
and explicitly stage only reviewed resolutions:

```bash
git status
git diff
# Edit the conflicted files and run gofmt on changed Go files.
git add <reviewed-files>
bash scripts/pinable/sync-upstream.sh --continue
```

If validation fails, fix and stage the change, then use `--continue` again.
No merge commit is created after a failed build or test. To abandon the pending
merge started by the script, run:

```bash
bash scripts/pinable/sync-upstream.sh --abort
```

The script enables `rerere.enabled=true` and `rerere.autoupdate=false` locally.
Git can suggest a recorded resolution for a recurring conflict, but it is never
auto-staged: inspect it and run the regression suite again. The rerere cache is
local to this clone, is not distributed merely by committing this script, and
cannot guarantee that future semantic conflicts will disappear.

After success, review the branch diff, push the named sync branch, and open a PR
to the fork's `main`. Merge using **Create a merge commit**. The read-only
`Pinable upstream check` workflow validates the actual PR merge result, builds
the server, runs the Go suite, checks the script, and probes version/discovery.

## Script regression tests

```bash
python3 scripts/pinable/test_sync_upstream.py -v
```

These tests use isolated temporary Git repositories and a fake Go command to
exercise merge ancestry, no-op synchronization, dirty/untracked-file refusal,
manual conflict continuation, abort, and failed-validation behavior. They test
the shell workflow, not the Go application; the CI Go suite remains required.
