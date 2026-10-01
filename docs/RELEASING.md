# Release procedure

> **Status:** draft. This procedure describes the gates for a future release;
> it is not evidence that v1.0, signed CLI artifacts, or visual compatibility
> fixtures already exist.

## 1. Confirm release scope

- Choose the version according to Semantic Versioning. Compatibility guarantees
  begin with v1.0.
- Confirm every release-blocking roadmap item is complete. Do not check a
  roadmap item merely because its documentation scaffold exists.
- For v1.0, complete the API review and breaking-change ledger in
  [`MIGRATION.md`](MIGRATION.md).
- Define the CLI operating-system and architecture targets for the release.

## 2. Prepare documentation

- Move relevant entries from `CHANGELOG.md` under a dated version heading.
- Update `README.md` and `README_AR.md` together.
- Record tested client versions and dates in `COMPATIBILITY.md`.
- Confirm security reporting and the supported-version policy in `SECURITY.md`.
- Check every documentation link and example command.

## 3. Run automated quality gates

Run the same core commands documented for contributors:

```bash
go fmt ./...
go vet ./...
go test ./...
go test -shuffle=on -count=3 ./...
go test -race ./...
go test -run '^$' -bench . -benchmem ./...
```

Run every fuzz target for a release-appropriate duration, not only the short CI
smoke duration. Confirm the package-specific coverage gates pass in CI on
Linux, Windows, and macOS.

## 4. Verify client compatibility

Render the public synthetic compatibility fixtures and inspect them in the
declared Microsoft Word, LibreOffice, and Google Docs versions. Record:

- client version, operating system, and test date;
- fixture input and output identity;
- reviewer and comparison method;
- expected differences and any regression decision.

The structural Go suite is necessary but does not replace this gate.

## 5. Build CLI artifacts

Build from the intended release commit with a reproducible command that embeds
the version in the `main.version` variable, for example:

```bash
go build -trimpath -ldflags="-s -w -X main.version=v1.0.0" ./cmd/namat
```

Use the actual release version, record the Go toolchain version, and produce an
artifact manifest covering every declared target. Verify each binary reports
the intended version.

## 6. Checksum, sign, and verify

- Generate a cryptographic checksum manifest for all release artifacts.
- Sign the artifacts or manifest with the project's approved signing identity.
- Publish provenance sufficient to associate artifacts with the release commit
  and build process.
- Verify the signature and checksums using the documented end-user procedure
  before publishing.

The tag-triggered `.github/workflows/release.yml` workflow builds six
CGO-disabled archives with Go 1.27.1, writes `checksums.txt`, generates GitHub
artifact attestations, and creates the GitHub release. Verify a downloaded
archive with:

```bash
sha256sum --check checksums.txt --ignore-missing
gh attestation verify namat_v1.0.0_linux_amd64.tar.gz \
  --repo nawafinity/go-namat
```

The signed-release roadmap item remains incomplete until this workflow has run
successfully for a release candidate, its identity and recovery policy have
been reviewed, and every published target has been downloaded and verified.

## 7. Tag and publish

1. Confirm the release commit is clean and has passed every required gate.
2. Create the matching version tag only after the changelog and migration guide
   are final.
3. Publish the source release, artifact manifest, checksums, signatures,
   provenance, and verification instructions together.
4. Verify installation and `namat version` from each published artifact.
5. Announce known limitations and link the compatibility, migration, and
   security documents.

If an artifact or signature is invalid, stop distribution, mark the release as
affected, preserve the evidence, and publish a corrected version rather than
silently replacing immutable release assets.
