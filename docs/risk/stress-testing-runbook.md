# Stress Testing Runbook

Use this runbook when validating portfolio downside sensitivity through the v2.8.0 risk analytics APIs.

## Run Built-In Scenarios

List scenarios:

```bash
curl http://localhost:8080/api/risk/scenarios \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "X-Tenant-ID: default-tenant"
```

Run the demo:

```bash
TOKEN=<jwt> ./scripts/demo-risk-analytics.sh --run
```

## Interpret Results

- `baselineValue` is the current portfolio value from supplied positions.
- `stressedValue` is the value after scenario shocks and haircuts.
- `pnlImpactPercent` shows the scenario loss or gain as a percentage of baseline.
- `worstScenario` is the scenario with the lowest PnL impact.
- `riskLevel` is based on deterministic local thresholds.
- `recommendations` are explainable next actions.

## Common Failure Cases

Missing positions:

- The API returns a zero baseline and zero PnL instead of dividing by zero.
- Treat this as an input issue unless the portfolio is genuinely empty.

Invalid scenario:

- `/api/v1/risk/scenarios/run` returns `400` when a scenario name is unknown.
- Confirm names against `GET /api/risk/scenarios`.

Negative quantity or price:

- Pydantic validation rejects negative quantities and prices.
- Fix upstream portfolio data before rerunning.

Extreme shock values:

- Stress APIs allow simple extreme values for demo analysis.
- Interpret very large shocks as exploratory, not calibrated forecasts.

Zero portfolio value:

- Percentage impact is returned as `0` to avoid unsafe division.
- Check whether all current prices or quantities are missing.

## Correlation

Pass `X-Correlation-ID` to connect risk analytics calls with gateway logs, Kafka events, audit records, and traces:

```bash
curl -X POST http://localhost:8080/api/risk/stress-test \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "X-Tenant-ID: default-tenant" \
  -H "X-Correlation-ID: demo-correlation-123" \
  -H "Content-Type: application/json" \
  --data @docs/examples/risk/stress-test-request.json
```

Generated events include `tenantId`, `correlationId`, `portfolioId`, `riskLevel`, and timestamp fields. Use the correlation ID with logs, DLQ records, audit APIs, and Jaeger traces.
