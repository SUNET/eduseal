# eduSeal

## docker release version

`testing` tracks the latest release tag and is built from branch `main`.
`prod` is promoted from a specific version — no rebuild, just a re-tag.

See [docs/RELEASE.md](docs/RELEASE.md) for the full release process.

### Quick reference

```bash
# Create a new release (patch bump by default)
make release

# Bump minor or major
make release BUMP=minor
make release BUMP=major

# Promote to production
make release-prod              # promotes the latest version
make release-prod TAG=v1.2.3   # promotes a specific version
```

## branches

`main` is the stable development branch.

## How to build

### Docker

For convenience all services are build within a docker container.

Each service has its own make target, `make docker-build-<service>` or build all of them at once `make docker-build`

### Static binary without Docker

run `make build-<service>` to build a specific service or `make build` to build all services.

linux/amd64 is consider supported, other build options may work.

## Start, Stop & Restart

`make start` or `docker-compose -f docker-compose.yaml up -d --remove-orphans`

`make stop` or `docker-compose -f docker-compose.yaml rm -s -f`

`make restart`

## Swagger

### Endpoint

`GET http://<apigw-url>/swagger/doc.json`

or with web browser: `http://<apigw-url>/swagger/index.html`
