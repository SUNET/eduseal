# Release Process

## Overview

All services (**apigw**, **sealer**, **validator**) are always released together under a single shared version tag. A release creates an annotated git tag and pushes it to origin, then images are built and pushed from a trusted local environment using `make local-publish`.

### Tag format

```text
v<major>.<minor>.<patch>
```

Examples: `v1.2.3`, `v2.0.0`, `v1.0.1`

---

## Quick Start

```bash
# Bump patch (default):  v1.0.0 -> v1.0.1
make release

# Bump minor:  v1.0.0 -> v1.1.0
make release BUMP=minor

# Bump major:  v1.0.0 -> v2.0.0
make release BUMP=major
```

## What Happens

1. **Finds** the latest `vX.Y.Z` git tag.
2. **Bumps** the version according to `BUMP` (major / minor / patch).
3. **Checks** the working tree is clean (no uncommitted changes).
4. **Checks** the new tag doesn't already exist.
5. **Creates** an annotated git tag (`vX.Y.Z`).
6. **Pushes** the tag to `origin`.
7. **Locally publishes** release images automatically as part of `make release`:
   - Builds Docker images for **all three** services.
   - Pushes images to `docker.sunet.se/eduseal/<image>:<version>` and `:testing`.

## Pre-requisites

- All changes must be committed and pushed before releasing.
- You must have push access to the repository.
- You must be logged in to `docker.sunet.se` from your local environment (`docker login docker.sunet.se`).

## Local Publish

Release images are published from your local environment using:

```bash
make release
```

This target:
- validates `VERSION`
- builds `apigw`, `sealer_lunahsm`, `validator`
- pushes versioned tags and `:testing`

Images built:
- `docker.sunet.se/eduseal/apigw` from `docker/apigw/Dockerfile`
- `docker.sunet.se/eduseal/sealer_lunahsm` from `docker/sealer/lunahsm/Dockerfile`
- `docker.sunet.se/eduseal/validator` from `docker/validator/Dockerfile`

## Docker Images Produced

Each release pushes two tags per service: versioned and `testing`.
Production images use two tags, promoted separately via `make release-prod`:
- `prod`
- `prod-vX.Y.Z`

| Service   | Versioned                                           | Testing                                          | Production                                        |
| --------- | --------------------------------------------------- | ------------------------------------------------ | ------------------------------------------------- |
| apigw     | `docker.sunet.se/eduseal/apigw:<version>`           | `docker.sunet.se/eduseal/apigw:testing`          | `docker.sunet.se/eduseal/apigw:prod`              |
| sealer    | `docker.sunet.se/eduseal/sealer_lunahsm:<version>`  | `docker.sunet.se/eduseal/sealer_lunahsm:testing` | `docker.sunet.se/eduseal/sealer_lunahsm:prod`     |
| validator | `docker.sunet.se/eduseal/validator:<version>`        | `docker.sunet.se/eduseal/validator:testing`       | `docker.sunet.se/eduseal/validator:prod`           |

## Examples

```bash
# Bump patch (e.g. latest tag is v1.5.0)
make release
# → Detects v1.5.0, creates tag: v1.5.1
# → Then automatically builds/pushes images locally at v1.5.1 + testing

# Bump major (e.g. latest tag is v1.5.1)
make release BUMP=major
# → Detects v1.5.1, creates tag: v2.0.0

# Promote latest version to prod (no rebuild — local re-tag/push)
make release-prod
# → Locally pulls :v1.5.1 images, re-tags as :prod and :prod-v1.5.1, pushes

# Promote a specific version to prod
make release-prod TAG=v1.2.3
# → Locally pulls :v1.2.3 images, re-tags as :prod and :prod-v1.2.3, pushes
```

## Troubleshooting

| Error                                 | Cause                  | Fix                                |
| ------------------------------------- | ---------------------- | ---------------------------------- |
| `BUMP must be major, minor, or patch` | Invalid bump type      | Use `BUMP=major`, `minor`, `patch` |
| `tag already exists`                  | Version already exists | Bump again or check tags           |
| `working tree is dirty`               | Uncommitted changes    | `git add . && git commit` first    |
| `not found in registry`               | Image tag missing      | Run `make release` first           |
