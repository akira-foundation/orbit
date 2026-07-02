# Orbit Bundled Services — MinIO (Phase 3) Design

Date: 2026-07-02
Status: Approved

## Goal

Ship MinIO as the third bundled service: S3-compatible object storage with a
bucket per project and AWS-style credentials injected at dev-server spawn,
plus the MinIO console proxied at `s3.orbit.test`.

## Binary source (verified by real fetch)

GitHub releases, tag `RELEASE.2025-09-07T16-13-09Z`. Assets are **raw
binaries** (no archive) named `minio.<platform>.<tag>`, one per platform,
each with a published `.sha256sum`. All four platforms verified (HTTP 200 +
checksum fetched): darwin-arm64, darwin-amd64, linux-arm64, linux-amd64.

Pinned SHA256:

| Platform | SHA256 |
|----------|--------|
| darwin-arm64 | 7c3b3039b76e55a1b80935848ed83998d5e8d317374f87851f46a019ff5c0aa4 |
| darwin-amd64 | 4759080aeef7385aaceaac1131c30aaeb99605921553967dc6d3ef4e16ac64f9 |
| linux-arm64 | 5c83cd2cf151717ba0243f73e1c7802ff36e272b67144bdd7f1f7d684fd6f03d |
| linux-amd64 | 7c5bd8512c6e966455b1d198209358b2d191c77a83ab377c4073281065fb855f |

## Decisions

| Topic | Decision |
|-------|----------|
| Acquisition | New `Engine.RawBinary` mode in the acquirer: download, verify pinned SHA256, save + chmod — no extraction. |
| Instance | One shared instance. API on `127.0.0.2:9000` (`Engine.Port`), console on `127.0.0.2:9001` (`WebPort`, proxied at `s3.orbit.test` — existing service table picks it up via `WebDomain: "s3"`). |
| Root credentials | Fixed local-dev pair `orbit` / `orbitsecret`, passed as `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD` in the spawned process env. Local-only bind — same threat model as Postgres trust auth. New `Engine.Env []string` field carries them. |
| Start args | `server <dataDir> --address 127.0.0.2:9000 --console-address 127.0.0.2:9001` |
| Provisioning | Bucket per project named `slug`, via `minio-go/v7` (new Go dep): `BucketExists` → `MakeBucket`, idempotent. `Engine.Provision` hook, same as Postgres. |
| Injected env (absent-only) | `AWS_ACCESS_KEY_ID=orbit`, `AWS_SECRET_ACCESS_KEY=orbitsecret`, `AWS_DEFAULT_REGION=us-east-1`, `AWS_BUCKET=<slug>`, `AWS_ENDPOINT=http://127.0.0.2:9000`, `AWS_URL=http://127.0.0.2:9000/<slug>`, `AWS_USE_PATH_STYLE_ENDPOINT=true` |
| UI | Plain card (no family). Setup dialog: connection fields + snippets (.env generic, Laravel filesystems, Node aws-sdk). No frontend changes needed beyond catalog-provided Setup content. |
| Slug caveat | S3 bucket names forbid underscores; project slugs are kebab-case already (Slugify), so slugs are valid bucket names as-is. |

## Testing

- Unit: raw-binary acquirer path (httptest), catalog entry shape, MinioEnv map.
- Integration (network-gated, `-short` skips): acquire real MinIO, start on
  127.0.0.2, provision bucket, idempotent re-provision, stop.

## Out of scope

Valkey (blocked on macOS binary source), per-project access keys, bucket
policies/versioning.
