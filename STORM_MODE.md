# Storm Mode - Continuous Testing

Storm mode is a continuous testing feature for the noctool that allows you to stress test the EduSeal API with configurable parameters.

## Configuration

Add the following configuration to your `noctool_config.yaml`:

```yaml
storm:
  enabled: true                # Enable/disable storm mode
  max_retries: 5                # Number of fetch attempts per upload
  retry_wait_ms: 500           # Wait time between fetch retries (ms)
  upload_interval_ms: 2000     # Wait time between uploads (ms)
  max_uploads: 10              # Maximum number of uploads (ignored if continuous: true)
  continuous: false             # Set to true for infinite testing
  fetch_timeout_sec: 15        # Timeout for fetching sealed PDF (seconds)
```

## Configuration Parameters

### `enabled` (boolean)
- **Default:** `false`
- **Description:** Enables or disables storm mode. When enabled, noctool will run continuous testing instead of a single test.

### `max_retries` (integer)
- **Default:** `3`
- **Description:** Number of attempts to fetch the signed PDF for each upload. If the PDF is not ready immediately, the system will retry this many times.

### `retry_wait_ms` (integer)
- **Default:** `500`
- **Description:** Wait time in milliseconds between fetch retry attempts. Lower values mean more aggressive polling.

### `upload_interval_ms` (integer)
- **Default:** `1000`
- **Description:** Wait time in milliseconds between each PDF upload cycle. This controls the rate of continuous uploads.

### `max_uploads` (integer)
- **Default:** `10`
- **Description:** Maximum number of PDF uploads to perform. Set to `0` for unlimited/continuous testing.

### `fetch_timeout_sec` (integer)
- **Default:** `11`
- **Description:** Maximum time in seconds to wait for a signed PDF to be available before timing out.

## Usage Examples

### Example 1: Basic Load Testing (10 uploads)
```yaml
storm:
  enabled: true
  max_retries: 5
  retry_wait_ms: 500
  upload_interval_ms: 2000
  max_uploads: 10
  fetch_timeout_sec: 15
```

Run with:
```bash
./bin/noctool -config noctool_config.yaml
```

### Example 2: Continuous Stress Testing
```yaml
storm:
  enabled: true
  max_retries: 3
  retry_wait_ms: 300
  upload_interval_ms: 1000
  max_uploads: 0
  fetch_timeout_sec: 20
```

This will run continuously until manually stopped (Ctrl+C).

### Example 3: High-Frequency Testing
```yaml
storm:
  enabled: true
  max_retries: 10
  retry_wait_ms: 200
  upload_interval_ms: 500
  max_uploads: 50
  fetch_timeout_sec: 30
```

## Statistics

Storm mode tracks and displays comprehensive statistics:

- **Total Uploads**: Number of PDF upload attempts
- **Successful**: Number of successfully validated PDFs
- **Failed**: Number of failed uploads
- **Total Retries**: Cumulative number of fetch retry attempts
- **Success Rate**: Percentage of successful uploads
- **Elapsed Time**: Total time since storm mode started

Statistics are displayed after each upload cycle with a formatted table.

## Output Example

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚡ STORM MODE - Continuous Testing ⚡
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Max Retries: 5
  Retry Wait: 500ms
  Upload Interval: 2000ms
  Fetch Timeout: 15s
  Max Uploads: 10
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[Upload #1] Starting at 14:32:15
  [Upload #1] Sealing PDF...
  [Upload #1] Transaction ID: abc123...
  [Upload #1] Fetch attempt 1/5...
  [Upload #1] ✓ PDF fetched successfully
  [Upload #1] Validating...
  [Upload #1] ✓ Validation: Intact=true, Valid=true
✓ [Upload #1] Completed successfully

╔════════════════════════════════════════╗
║          STORM STATISTICS              ║
╠════════════════════════════════════════╣
║ Total Uploads:      1                  ║
║ Successful:         1                  ║
║ Failed:             0                  ║
║ Total Retries:      1                  ║
║ Success Rate:       100.00%            ║
║ Elapsed Time:       3s                 ║
╚════════════════════════════════════════╝
```

## Saved Files

When `save: true` is set in the config, storm mode saves PDFs with the naming pattern:
```
storm_{upload_number}_{transaction_id}.pdf
```

For example: `storm_5_abc123def456.pdf`

## Tips

1. **Start with low volumes**: Begin with `max_uploads: 10` to verify the system works
2. **Monitor resources**: Watch CPU, memory, and network usage during continuous testing
3. **Adjust intervals**: Increase `upload_interval_ms` if the service is getting overwhelmed
4. **Use continuous mode carefully**: Set a specific number for `max_uploads` rather than `0` for production testing to avoid infinite loops
5. **Save selectively**: Disable `save: true` for high-volume testing to avoid filling disk space
