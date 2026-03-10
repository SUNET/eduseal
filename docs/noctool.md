noctool — Manual
================

`noctool` is a command-line NOC (Network Operations Centre) testing tool for the EduSeal PDF signing and validation service. It authenticates against the SUNET OAuth endpoint, submits PDFs for sealing, retrieves the signed result, and validates the signature. It also supports a **storm mode** for continuous / stress testing.

---

Table of Contents
-----------------

1.	[Prerequisites](#prerequisites)
2.	[Building](#building)
3.	[Usage](#usage)
4.	[Flags](#flags)
5.	[Configuration file](#configuration-file)
	-	[Top-level fields](#top-level-fields)
	-	[oauth](#oauth)
	-	[storm](#storm)
6.	[Test cases](#test-cases)
7.	[PDF sizes](#pdf-sizes)
8.	[Environments](#environments)
9.	[Storm mode](#storm-mode)
10.	[Output and exit codes](#output-and-exit-codes)
11.	[TLS requirements](#tls-requirements)

---

Prerequisites
-------------

### Client certificate and key

`noctool` uses mutual TLS (mTLS) to authenticate against both the SUNET auth endpoint and the EduSeal API. You need a PEM-encoded client certificate and its matching private key. The following certificate types are trusted by the service:

-	SITHS e-id funktionscertifikat (Inera)
-	E-identitet för offentlig sektor — EFOS (Försäkringskassan)
-	ExpiTrust EID CA V4

Place the certificate and key on disk and reference them via `client_cert` and `client_cert_key` in the config file.

### OAuth / JWT access token

`noctool` obtains a short-lived JWT bearer token from the SUNET auth service before every run. The token request is sent as a POST to `/transaction` on the auth endpoint (see [Environments](#environments)).

Two authentication methods are supported:

**Client key** (used by noctool) — the `oauth.client.key` field in the config must match a key name registered in the EduSeal server configuration:

```json
{
  "client": {
    "key": "<key_name_in_service_config>"
  }
}
```

**Certificate fingerprint** (alternative) — the server can also identify the client by the SHA-256 fingerprint of the mTLS certificate:

```json
{
  "client": {
    "key": {
      "proof": "mtls",
      "cert#S256": "<fingerprint>"
    }
  }
}
```

Both the client key and the certificate must be provisioned by SUNET in advance.

### ISRG Root X1 certificate

The tool loads the Let's Encrypt root certificate from `/etc/ssl/certs/ISRG_Root_X1.pem` for server verification. It will exit immediately if this file is missing. On most Linux distributions this is included in the `ca-certificates` package.

---

Building
--------

```bash
make build-noctool  # produces bin/noctool
```

---

Usage
-----

```text
noctool -config <path-to-config.yaml> [flags]
```

All runtime behaviour is controlled by a YAML configuration file supplied with `-config`. There are no positional arguments.

---

Flags
-----

| Flag       | Type   | Description                                          |
|------------|--------|------------------------------------------------------|
| `-config`  | string | **Required.** Path to the YAML configuration file.   |
| `-version` | bool   | Print the git commit hash and build date, then exit. |

### Examples

```bash
# Run a single seal-and-validate cycle
./bin/noctool -config noctool_config.yaml

# Print version information
./bin/noctool -version
```

---

Configuration file
------------------

The configuration is a YAML file. An annotated example:

```yaml
oauth:
  access_token:
    - flags:
        - bearer
      access:
        - type: eduseal
  client:
    key: masv_test_3          # OAuth client key issued by SUNET

env: test                     # Environment: "test" or "prod"
testcase: ladok               # Test case to run (currently only "ladok")
save: true                    # Save the sealed PDF to disk after validation
client_cert: ./client.cert    # Path to mTLS client certificate (PEM)
client_cert_key: ./client.key # Path to matching private key (PEM)
pdf_size: small               # PDF size used in tests: small, medium, big

storm:
  enabled: false              # Enable storm / continuous testing mode
  max_retries: 5
  retry_wait_ms: 500
  upload_interval_ms: 2000
  max_uploads: 10
  fetch_timeout_sec: 15
```

### Top-level fields

| Field             | Type   | Required | Description                                                                                                                                                              |
|-------------------|--------|----------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `env`             | string | yes      | Target environment. `test` or `prod`.                                                                                                                                    |
| `testcase`        | string | yes      | Test case to execute. Currently only `ladok` is supported.                                                                                                               |
| `save`            | bool   | no       | When `true`, the sealed PDF is written to `<transaction_id>.pdf` in the working directory (mode `0600`). In storm mode files are named `storm_<n>_<transaction_id>.pdf`. |
| `client_cert`     | string | yes      | Filesystem path to the PEM-encoded mTLS client certificate.                                                                                                              |
| `client_cert_key` | string | yes      | Filesystem path to the PEM-encoded private key for the client certificate.                                                                                               |
| `pdf_size`        | string | no       | Size of the built-in test PDF. `small` (default), `medium`, or `big`.                                                                                                    |

### oauth

Controls the JWT access-token request sent to the SUNET auth endpoint.

| Field                                | Description                            |
|--------------------------------------|----------------------------------------|
| `oauth.access_token[].flags`         | Token flags, e.g. `bearer`.            |
| `oauth.access_token[].access[].type` | Requested access type, e.g. `eduseal`. |
| `oauth.client.key`                   | The OAuth client credential key.       |

The OAuth client key and certificate must be provisioned by SUNET in advance.

### storm

See [Storm mode](#storm-mode) below for full details.

---

Test cases
----------

| Name    | Description                                                                                                                                    |
|---------|------------------------------------------------------------------------------------------------------------------------------------------------|
| `ladok` | Seal a test PDF, fetch the signed result, validate the signature, and optionally save the file. This is the primary NOC health-check scenario. |

---

PDF sizes
---------

Three built-in test PDFs are embedded in the binary:

| Value    | Approx. size | Pages |
|----------|--------------|-------|
| `small`  | ~350 B       | 1     |
| `medium` | ~2 KB        | 3     |
| `big`    | ~10 KB       | 10    |

The PDFs contain the text "SUNET EduSeal test PDF" and are Base64-encoded internally.

---

Environments
------------

| `env` value | API base URL                        | Auth URL                                 |
|-------------|-------------------------------------|------------------------------------------|
| `test`      | `https://test-api.eduseal.sunet.se` | `https://auth-test.sunet.se/transaction` |
| `prod`      | `https://api.eduseal.sunet.se`      | `https://auth.sunet.se/transaction`      |

Any other value causes the tool to exit with an error.

---

Storm mode
----------

Storm mode executes the full seal → fetch → validate cycle in a loop, useful for load testing and detecting intermittent failures.

Enable it by setting `storm.enabled: true` in the config file.

### Storm configuration fields

| Field                | Type | Default | Description                                                                  |
|----------------------|------|---------|------------------------------------------------------------------------------|
| `enabled`            | bool | `false` | Activates storm mode.                                                        |
| `max_retries`        | int  | `3`     | Fetch attempts per upload before declaring failure.                          |
| `retry_wait_ms`      | int  | `500`   | Milliseconds to wait between fetch retry attempts.                           |
| `upload_interval_ms` | int  | `1000`  | Milliseconds to wait between upload cycles.                                  |
| `max_uploads`        | int  | `10`    | Total number of upload cycles. Set to `0` for unlimited (runs until Ctrl+C). |
| `fetch_timeout_sec`  | int  | `11`    | Seconds to keep polling for the signed PDF before timing out.                |

### Token refresh

In storm mode, `noctool` runs a background goroutine that automatically refreshes the OAuth token 2 minutes before it expires. If a refresh fails, the goroutine retries after 30 seconds.

### Error log

A timestamped error log is created automatically at the start of each storm run:

```text
storm_errors_20060102_150405.log
```

Each line records the timestamp, upload number, transaction ID, and error message. Additionally, exhausted fetch-retry events are flagged with `FETCH ATTEMPTS EXHAUSTED`.

### Statistics

After each upload cycle a statistics table is printed to stdout:

```text
╔════════════════════════════════════════╗
║          STORM STATISTICS              ║
╠════════════════════════════════════════╣
║ Total Uploads:      10               ║
║ Successful:         9                ║
║ Failed:             1                ║
║ Total Retries:      12               ║
║ Success Rate:       90.00%           ║
║ Elapsed Time:       25s              ║
╚════════════════════════════════════════╝
```

### Example: unlimited stress test

```yaml
storm:
  enabled: true
  max_retries: 3
  retry_wait_ms: 300
  upload_interval_ms: 1000
  max_uploads: 0          # unlimited
  fetch_timeout_sec: 20
```

```bash
./bin/noctool -config noctool_config.yaml
# Press Ctrl+C to stop
```

### Example: bounded load test (50 uploads)

```yaml
storm:
  enabled: true
  max_retries: 5
  retry_wait_ms: 500
  upload_interval_ms: 2000
  max_uploads: 50
  fetch_timeout_sec: 15
```

---

Output and exit codes
---------------------

`noctool` uses ANSI colour codes in its output:

| Symbol | Colour | Meaning                   |
|--------|--------|---------------------------|
| `✓`    | green  | Operation succeeded       |
| `✗`    | red    | Operation failed          |
| `⚠`    | yellow | Warning / non-fatal issue |
| `⚡`   | yellow | Storm mode indicator      |

**Exit codes:**

| Code | Meaning                                                       |
|------|---------------------------------------------------------------|
| `0`  | Success                                                       |
| `1`  | Failure (config error, auth error, seal/validate error, etc.) |

---

TLS requirements
----------------

`noctool` enforces mutual TLS (mTLS):

-	**Server verification** uses the ISRG Root X1 (Let's Encrypt root) certificate loaded from `/etc/ssl/certs/ISRG_Root_X1.pem`. The tool will exit if this file is not present.
-	**Client authentication** requires a valid certificate/key pair specified in `client_cert` / `client_cert_key`. These are provided by SUNET.
-	TLS versions 1.2 and 1.3 are accepted; older versions are rejected.
-	`InsecureSkipVerify` is always `false`.
