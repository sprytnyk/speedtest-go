---
name: release
description: Ship a new version of speedtest-go. Commits pending work, runs the checks, makes a signed tag on master and pushes it (CI then builds the archives and the Docker image), builds the same archives reproducibly on this machine with scripts/build-release.sh, checks they match CI's, signs SHA256SUMS in dist/, and writes the release notes there too. Use when the user asks to release, ship, or "ship it as usual".
argument-hint: '[version, e.g. 2.0.1; derived from commits if omitted]'
disable-model-invocation: true
---

# Release

Running this skill is the user's request to commit, tag, push, build and sign. Their global
rules still apply. Never force-push, retag or rewrite pushed history without asking first,
never add AI attribution to commits or tags, never bypass hooks, and keep IPs and secrets out
of release notes.

Stop and report back if any step fails. Don't work around a failure.

There is no version file and no release commit. The version comes from the tag:
`scripts/build-release.sh` and the Docker workflow pass it to `-X main.version=...`.

## 1. Pre-flight

- You must be on `master`, and `git fetch origin` must show `master` neither behind nor
  diverged.
- **Uncommitted work** gets committed first. Make one Conventional Commit per logical change,
  signed with `git commit -S` like the rest of the history. Never stage secret files, `dist/`
  or scratch files, and leave `.claude/` out unless the user says otherwise. If it's unclear
  whether a change belongs in the release, ask.
- **Go toolchain.** `go version` must match the `toolchain` line in `go.mod`. CI builds with
  that version, and the archives only come out byte-identical with the same toolchain. If they
  differ, stop and ask.
- Run the checks CI runs. All must pass:
  ```bash
  test -z "$(gofmt -l .)" || gofmt -l .
  go vet ./...
  go test -race ./...
  ```
  If `scripts/build-release.sh` changed since the last tag, also run
  `shellcheck scripts/build-release.sh`.
- **Vulnerabilities.** CI also runs `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`.
  That fetches and runs third-party code, so ask before running it here. If it reports
  standard library vulnerabilities fixed in a newer Go patch release, tell the user and ask
  whether to bump the `toolchain` line first (a `build: ...` commit, and the release notes
  then say which Go version it's built with).
- **README.** Compare `git log <last-tag>..HEAD` with `README.md` (Features, Installation,
  Configuration, Storing results, ISP lookup and privacy, Languages, Differences from the PHP
  backend). Also check that every setting added or changed is in `settings.toml` with a
  comment. Update anything user-visible that's missing in a `docs(readme): ...` commit. Only
  refresh screenshots in `docs/screenshots/` if the user asks.

## 2. Version

- Last tag: `git describe --tags --abbrev=0`.
- Use the version given in `$ARGUMENTS`. If there is none, derive it with semver from the
  commits since the last tag: a `!` or `BREAKING CHANGE` means major, any `feat` means
  minor, anything else means patch. Say which version you picked and why, in one line.
- `git status --porcelain --untracked-files=no` must be empty, so the build isn't `-dirty`.

## 3. Signed tag

The tag goes on `master`'s HEAD. It is annotated and signed, and its message is the version,
a blank line, and a short summary of the release in plain words, wrapped at 72 columns (see
`git cat-file -p v1.3.0`). Write the message to a file in the scratchpad:

```bash
git tag -s vX.Y.Z -F <scratchpad>/tag-message
git tag -v vX.Y.Z
```

The signature must be Good, from `0xC00C0F0D647BD3C6`. If gpg pinentry times out, ask the
user to run the command themselves with a `!` prefix.

## 4. Push

```bash
git push origin master
git push origin vX.Y.Z
```

If either push is rejected, stop and ask. Never force-push.

The tag push starts two workflows: *Release builds* (the archives, kept as a run artifact) and
*Docker* (`ghcr.io/sprytnyk/speedtest-go:vX.Y.Z` and `latest`).

## 5. Local build

`dist/` is gitignored, and `dist/vX.Y.Z` must not exist yet:

```bash
scripts/build-release.sh dist/vX.Y.Z
```

The version comes from the tag. Run it with `run_in_background`, sending its output to a log
file in the scratchpad, and wait for the notification. It takes about 5 minutes. Don't poll.

Then check:

- `dist/vX.Y.Z` holds 13 archives (10 `.tar.gz`, 3 `.zip`) and `SHA256SUMS`, all named
  `speedtest-go_X.Y.Z_*`.
- The version is baked in:
  ```bash
  tar -xzOf dist/vX.Y.Z/speedtest-go_X.Y.Z_linux_amd64.tar.gz speedtest-backend > <scratchpad>/speedtest-backend
  chmod +x <scratchpad>/speedtest-backend && <scratchpad>/speedtest-backend -version
  ```
  It must print `speedtest-go X.Y.Z`.

## 6. Compare with CI

The builds are reproducible, so CI's archives must have the same checksums as the local ones.
`gh` isn't installed, so ask the user to:

- confirm that the *Release builds* and *Docker* runs for `vX.Y.Z` passed, and
- download the `speedtest-go_X.Y.Z` artifact of the *Release builds* run and either paste its
  `SHA256SUMS` or say where they unzipped it.

Then `diff` it against `dist/vX.Y.Z/SHA256SUMS`. If they differ, stop and report which
archives differ and the `go version` of both builds (the top of the CI log and of the local
log). If the user skips this step, say so in the report.

## 7. Sign the checksums

From `dist/vX.Y.Z`, sign and verify:

```bash
gpg --batch --yes --local-user 0xC00C0F0D647BD3C6 --armor --detach-sign SHA256SUMS
gpg --verify SHA256SUMS.asc SHA256SUMS
sha256sum -c SHA256SUMS
```

The signature must say Good, from `0xC00C0F0D647BD3C6`.

## 8. Release notes

Write them from the commits since the previous tag, in plain words, and say why a change
matters when it isn't obvious. Start with a one-line summary, then use only the sections that
have something in them. Save them to `dist/vX.Y.Z/RELEASE_NOTES.md` (the content below,
without the fence), next to the archives and checksums. They aren't in `SHA256SUMS`. Also
give them in one fenced `markdown` block in the report, ready to paste into the GitHub
release:

````markdown
One-line summary of the release.

### Breaking
- What changed, why, and what users have to do (settings, links, database).

### Highlights
- New features, user-visible first.

### Fixes
- What was broken and what happens now. Mention the Go version if the toolchain changed.

### Changes
- Build, CI, packaging and dependency changes users may notice.

### Removed
- Features, files, settings or build targets that are gone.

### Docker
`docker pull ghcr.io/sprytnyk/speedtest-go:vX.Y.Z`

### Verify
`gpg --verify SHA256SUMS.asc SHA256SUMS && sha256sum --ignore-missing -c SHA256SUMS`
````

## 9. Report

Report briefly:

- the version and tag, and that the tag signature is Good
- the commits that went in
- that the local archives match CI's checksums (or that the comparison was skipped)
- the files in `dist/vX.Y.Z/` (13 archives, `SHA256SUMS`, `SHA256SUMS.asc`,
  `RELEASE_NOTES.md`) and the verification result
- the release notes block
- what's left for the user: create the GitHub release for `vX.Y.Z`, upload the 13 archives,
  `SHA256SUMS` and `SHA256SUMS.asc` from `dist/vX.Y.Z/`, and paste the notes
- leftovers, such as the extracted binary in the scratchpad or skipped checks
