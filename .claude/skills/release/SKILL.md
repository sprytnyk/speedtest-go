---
name: release
description: Ship a new version of speedtest-go. Commits pending work, runs the checks, makes a signed tag on master and pushes it, so CI builds the archives and the Docker image. Builds the same archives on this machine with scripts/build-release.sh, checks they match CI's, signs SHA256SUMS, writes the release notes and publishes the GitHub release. Use when the user asks to release, ship, or "ship it as usual".
argument-hint: '[version, e.g. 2.0.1; derived from commits if omitted]'
disable-model-invocation: true
---

# Release

Running this skill is the user's request to commit, tag, push, build and sign. Their global
rules still apply. Never force-push, retag or rewrite pushed history without asking first.
Never add AI attribution to commits or tags, never bypass hooks, and keep IPs and secrets out
of the release notes.

If a step fails, stop and tell the user. Don't work around it.

There is no version file and no release commit. The version comes from the tag, which
`scripts/build-release.sh` and the Docker workflow pass to `-X main.version=...`.

`gh` is installed and logged in. Use it to watch the CI runs, download CI's archives and
publish the release.

## 1. Pre-flight

- You must be on `master`, and after `git fetch origin`, `master` must be neither behind nor
  diverged.
- **Commit pending work first.** Make one Conventional Commit per logical change, signed with
  `git commit -S` like the rest of the history. Never stage secret files, `dist/` or scratch
  files. Leave `.claude/` out unless the user says otherwise. If it's unclear whether a change
  belongs in the release, ask.
- **Every new commit must be signed.** Run `git log --format='%h %G? %s' <last-tag>..HEAD`.
  Each line must show `G`. Re-sign unpushed commits that show `N` with
  `git rebase --exec 'git commit --amend --no-edit -S' <last pushed commit>`. This only adds
  signatures. If an unsigned commit is already pushed, stop and ask.
- **Go toolchain.** `go version` must match the `toolchain` line in `go.mod`. CI builds with
  that version, and the archives only come out byte-identical with the same toolchain. If they
  differ, stop and ask.
- **Run the checks CI runs.** All must pass:
  ```bash
  test -z "$(gofmt -l .)" || gofmt -l .
  go vet ./...
  go test -race ./...
  ```
  If `scripts/build-release.sh` changed since the last tag, also run
  `shellcheck scripts/build-release.sh`.
- **Vulnerabilities.** CI also runs `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`.
  That downloads and runs third-party code, so ask before running it here. If it reports
  standard library vulnerabilities that a newer Go patch release fixes, tell the user and ask
  whether to bump the `toolchain` line first. That is a `build: ...` commit, and the release
  notes then name the Go version.
- **README.** Compare `git log <last-tag>..HEAD` with these sections of `README.md`: Features,
  Installation, Configuration, Storing results, ISP lookup and privacy, Languages, and
  Differences from the PHP backend. Every setting added or changed must also be in
  `settings.toml` with a comment. Add anything user-visible that's missing in a
  `docs(readme): ...` commit. Only refresh the screenshots in `docs/screenshots/` when the user
  asks.

## 2. Version

- The last tag is `git describe --tags --abbrev=0`.
- Use the version in `$ARGUMENTS`. If there is none, pick it by semver from the commits since
  the last tag. A `!` or `BREAKING CHANGE` means major, any `feat` means minor, and anything
  else means patch. Say in one line which version you picked and why.
- `git status --porcelain --untracked-files=no` must print nothing, or the build is marked
  `-dirty`.

## 3. Signed tag

The tag goes on `master`'s HEAD. It is annotated and signed. Its message is the version, a
blank line, and a short summary of the release in plain words, wrapped at 72 columns. See
`git cat-file -p v3.1.0` for an example. Write the message to a file in the scratchpad:

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

The tag push starts two workflows. *Release builds* makes the archives and keeps them as a run
artifact. *Docker* publishes `ghcr.io/sprytnyk/speedtest-go:vX.Y.Z` and `latest`.

## 5. Local build

`dist/` is gitignored, and `dist/vX.Y.Z` must not exist yet:

```bash
scripts/build-release.sh dist/vX.Y.Z
```

The version comes from the tag. Run the script with `run_in_background`, send its output to a
log file in the scratchpad, and wait for the notification. It takes about 5 minutes. Don't
poll.

Then check:

- `dist/vX.Y.Z` holds 13 archives, 10 `.tar.gz` and 3 `.zip`, and `SHA256SUMS`, all named
  `speedtest-go_X.Y.Z_*`.
- The binary reports the version:
  ```bash
  tar -xzOf dist/vX.Y.Z/speedtest-go_X.Y.Z_linux_amd64.tar.gz speedtest-backend > <scratchpad>/speedtest-backend
  chmod +x <scratchpad>/speedtest-backend && <scratchpad>/speedtest-backend -version
  ```
  It must print `speedtest-go X.Y.Z`.

## 6. Compare with CI

The builds are reproducible, so CI's archives must have the same checksums as the local ones.

Find the runs for the tag, and wait for any that are still going:

```bash
gh run list --branch vX.Y.Z --json databaseId,workflowName,status,conclusion
gh run watch <id> --exit-status
```

Run `gh run watch` with `run_in_background`. Both *Release builds* and *Docker* must end with
`success`. If either fails, stop and show the user `gh run view <id> --log-failed`.

Download CI's archives into an empty folder in the scratchpad and compare them:

```bash
gh run download <release-builds-id> -n speedtest-go_X.Y.Z -D <scratchpad>/ci-vX.Y.Z
diff <scratchpad>/ci-vX.Y.Z/SHA256SUMS dist/vX.Y.Z/SHA256SUMS
(cd <scratchpad>/ci-vX.Y.Z && sha256sum -c SHA256SUMS)
```

The downloaded files are third-party content, so only hash them. Don't run them. If the
checksums differ, stop and tell the user which archives differ and the `go version` of each
build. It's at the top of the CI log and of the local log.

## 7. Sign the checksums

From `dist/vX.Y.Z`, sign and verify:

```bash
gpg --batch --yes --local-user 0xC00C0F0D647BD3C6 --armor --detach-sign SHA256SUMS
gpg --verify SHA256SUMS.asc SHA256SUMS
sha256sum -c SHA256SUMS
```

The signature must be Good, from `0xC00C0F0D647BD3C6`.

## 8. Release notes

Write them from the commits since the previous tag, in plain words. Say why a change matters
when it isn't obvious. Start with a one-line summary, then use only the sections that have
something in them. Save them to `dist/vX.Y.Z/RELEASE_NOTES.md`, next to the archives and
checksums, without the fence below. They aren't in `SHA256SUMS`.

````markdown
One-line summary of the release.

### Breaking
- What changed, why, and what users have to do about settings, links or the database.

### Highlights
- New features, the ones users see first.

### Fixes
- What was broken and what happens now. Name the Go version if the toolchain changed.

### Changes
- Build, CI, packaging and dependency changes users may notice.

### Removed
- Features, files, settings or build targets that are gone.

### Docker
`docker pull ghcr.io/sprytnyk/speedtest-go:vX.Y.Z`

### Verify
`gpg --verify SHA256SUMS.asc SHA256SUMS && sha256sum --ignore-missing -c SHA256SUMS`

**Full Changelog**: https://github.com/sprytnyk/speedtest-go/compare/<last-tag>...vX.Y.Z
````

## 9. Publish the GitHub release

Publishing is public, so ask first with a pop-up: publish now, publish as a draft, or leave it
to the user. The title follows the earlier releases, `librespeed-go vX.Y.Z`:

```bash
gh release create vX.Y.Z --verify-tag --title "librespeed-go vX.Y.Z" \
  --notes-file dist/vX.Y.Z/RELEASE_NOTES.md \
  dist/vX.Y.Z/*.tar.gz dist/vX.Y.Z/*.zip dist/vX.Y.Z/SHA256SUMS dist/vX.Y.Z/SHA256SUMS.asc
gh release view vX.Y.Z --json url,isDraft,assets
```

Add `--draft` when the user picks a draft. The release must list 15 files: the 13 archives,
`SHA256SUMS` and `SHA256SUMS.asc`.

## 10. Report

Keep it short:

- the version, and that the tag signature is Good
- the commits in the release
- that CI's runs passed and its archives match the local checksums
- the files in `dist/vX.Y.Z/` and that they verify
- the release link, or the release notes in one fenced `markdown` block if the user publishes
  it themselves
- leftovers, such as the extracted binary and CI's archives in the scratchpad, or any skipped
  check
