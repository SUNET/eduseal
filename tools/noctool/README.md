# noctool

`noctool` is a command-line NOC (Network Operations Centre) testing tool for the EduSeal PDF signing and validation service. It authenticates against the SUNET OAuth endpoint, submits PDFs for sealing, retrieves the signed result, and validates the signature. It also supports a **storm mode** for continuous/stress testing.

## Building

```bash
make build-noctool  # produces bin/noctool
```

## Usage

```bash
noctool -config <path-to-config.yaml>
```

### Flags

| Flag       | Type   | Description                                          |
|------------|--------|------------------------------------------------------|
| `-config`  | string | **Required.** Path to the YAML configuration file.   |
| `-version` | bool   | Print the git tag, commit hash, and build date, then exit. |

## Configuration

See `noctool_config.yaml` in the repository root for an annotated example. Key fields:

| Field             | Description                                              |
|-------------------|----------------------------------------------------------|
| `env`             | Target environment: `test` or `prod`                     |
| `testcase`        | Test case to run (currently only `ladok`)                |
| `save`            | Save the sealed PDF to disk after validation             |
| `client_cert`     | Path to PEM-encoded mTLS client certificate              |
| `client_cert_key` | Path to matching private key                             |
| `pdf_size`        | Size of built-in test PDF: `small`, `medium`, or `big`   |
| `oauth`           | OAuth client key and token request configuration         |
| `storm`           | Storm mode settings (see below)                          |

## Storm Mode

Storm mode executes the full seal → fetch → validate cycle in a loop for load testing.

Enable with `storm.enabled: true` in the config file.

| Field                | Default | Description                                         |
|----------------------|---------|-----------------------------------------------------|
| `max_retries`        | 3       | Fetch attempts per upload before declaring failure   |
| `retry_wait_ms`      | 500     | Milliseconds between fetch retries                  |
| `upload_interval_ms` | 1000    | Milliseconds between upload cycles                  |
| `max_uploads`        | 10      | Total upload cycles (`0` = unlimited, Ctrl+C stops) |
| `fetch_timeout_sec`  | 11      | Seconds to poll for the signed PDF                  |

## Prerequisites

- **Client certificate**: mTLS cert/key pair provisioned by SUNET
- **ISRG Root X1**: Must be present at `/etc/ssl/certs/ISRG_Root_X1.pem`
- **OAuth client key**: Registered in the EduSeal server configuration
