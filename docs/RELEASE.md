# Release Process

## Overview

All services (**apigw**, **sealer**, **validator**) are always released together under a single shared version tag. A release creates an annotated git tag and pushes it to origin, which triggers a Jenkins pipeline that builds and pushes Docker images for all three services.

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
7. **Jenkins** detects the new tag and automatically:
   - Builds Docker images for **all three** services in parallel.
   - Pushes images to `docker.sunet.se/eduseal/<image>:<version>` and `:testing`.

## Pre-requisites

- All changes must be committed and pushed before releasing.
- You must have push access to the repository.
- Jenkins must be configured to trigger on tag push events (see below).

## Jenkins Setup

The [.jenkins.yaml](../.jenkins.yaml) in the repo root defines the pipeline using the SUNET JJB format. The job uses a `script` builder that runs `docker build` and `docker push` for all three services.

The job is triggered by GitHub push events via `github_push`.

The build script only builds/pushes images when a release tag is detected (`vX.Y.Z` or `prod-vX.Y.Z`), and exits early for non-tag pushes.

For a release, the important event is the **tag push** created by `make release`.

Images built:
- `docker.sunet.se/eduseal/apigw` from `docker/apigw/Dockerfile`
- `docker.sunet.se/eduseal/sealer_lunahsm` from `docker/sealer/lunahsm/Dockerfile`
- `docker.sunet.se/eduseal/validator` from `docker/validator/Dockerfile`

## Docker Images Produced

Each release pushes two tags per service: the versioned tag and `testing` (always points to the latest release).
Production images use the `prod` tag, promoted separately via `make release-prod`.

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
# → Jenkins builds all three images at v1.5.1 + tags them as testing

# Bump major (e.g. latest tag is v1.5.1)
make release BUMP=major
# → Detects v1.5.1, creates tag: v2.0.0

# Promote testing to prod (no rebuild — Jenkins re-tags existing images)
make release-prod
# → Pushes git tag: prod-v1.5.1
# → Jenkins pulls :v1.5.1 images, re-tags as :prod, pushes

# Promote a specific version to prod
make release-prod TAG=v1.2.3
# → Pushes git tag: prod-v1.2.3
# → Jenkins pulls :v1.2.3 images, re-tags as :prod, pushes
```

## Troubleshooting

| Error                                 | Cause                  | Fix                                |
| ------------------------------------- | ---------------------- | ---------------------------------- |
| `BUMP must be major, minor, or patch` | Invalid bump type      | Use `BUMP=major`, `minor`, `patch` |
| `tag already exists`                  | Version already exists | Bump again or check tags           |
| `working tree is dirty`               | Uncommitted changes    | `git add . && git commit` first    |
| `not found in registry`               | Image tag missing      | Run `make release` first           |
