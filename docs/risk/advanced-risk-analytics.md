# Advanced Risk Analytics

v2.8.0 extends `risk-engine-service` with deterministic portfolio stress testing, scenario analysis, concentration risk, drawdown trend analysis, volatility shock simulation, and structured risk recommendations.

The implementation is intentionally explainable: it uses supplied positions and simple formulas rather than real market history or heavy financial libraries.

## APIs

Base service path: `/api/v1/risk`

Gateway path: `/api/risk`

| Method | Gateway path | Purpose |
| --- | --- | --- |
| `GET` | `/api/risk/scenarios` | List built-in scenarios. |
| `POST` | `/api/risk/stress-test` | Run supplied scenarios against supplied positions. |
| `POST` | `/api/risk/scenarios/run` | Run named built-in scenarios against supplied positions. |
| `GET` | `/api/risk/portfolio/{portfolioId}/concentration` | Demo/future integration concentration view. |
| `POST` | `/api/risk/portfolio/concentration` | Calculate concentration from supplied positions. |
| `GET` | `/api/risk/portfolio/{portfolioId}/drawdown-trend` | Demo/future integration drawdown view. |
| `POST` | `/api/risk/portfolio/drawdown-trend` | Calculate drawdown from supplied historical values. |
| `POST` | `/api/risk/volatility-shock` | Simulate volatility multiplier impact. |

All endpoints preserve `X-Tenant-ID` and `X-Correlation-ID` in responses. If absent, tenant defaults to `default-tenant` and the service generates a correlation ID.

## Formulas

Baseline value:

```text
sum(quantity * currentPrice)
```

Scenario price shock:

```text
stressedPrice = currentPrice * (1 + shockPercent / 100)
```

Shock precedence is symbol, then sector, then market. Liquidity haircut applies after price shocks.

PnL:

```text
pnlImpact = stressedValue - baselineValue
pnlImpactPercent = pnlImpact / baselineValue * 100
```

Concentration score:

```text
max(topSymbolExposurePercent, topSectorExposurePercent)
```

Drawdown:

```text
drawdownPercent = (currentValue - runningPeak) / runningPeak * 100
```

Volatility shock:

```text
shockedRiskScore = baseRiskScore * volatilityMultiplier
stressedValue = baselineValue * (1 - min(0.5, 0.05 * volatilityMultiplier))
```

## Risk Levels

Concentration:

| Score | Level |
| --- | --- |
| `< 25` | `LOW` |
| `25-39.999` | `MEDIUM` |
| `40-60` | `HIGH` |
| `> 60` | `CRITICAL` |

Drawdown:

| Max drawdown | Level |
| --- | --- |
| `<= 5%` | `LOW` |
| `> 5% and <= 10%` | `MEDIUM` |
| `> 10% and <= 20%` | `HIGH` |
| `> 20%` | `CRITICAL` |

Scenario loss uses the same practical bands: `LOW` up to 5%, `MEDIUM` over 5%, `HIGH` over 10%, and `CRITICAL` over 20%.

## Recommendations

Recommendations are deterministic:

- High concentration generates `REDUCE_CONCENTRATION`.
- Scenario loss over 10% generates `MARKET_DOWNSIDE_SENSITIVITY`.
- Drawdown over 20% generates `REVIEW_DRAWDOWN_CONTROLS`.
- Volatility multiplier greater than or equal to 2 generates `REDUCE_VOLATILITY_EXPOSURE`.

## Metrics

| Metric | Purpose |
| --- | --- |
| `risk_stress_tests_total{status}` | Stress test request count. |
| `risk_scenarios_run_total{scenario,status}` | Named scenario run count. |
| `risk_concentration_analyses_total{status}` | Concentration analysis count. |
| `risk_drawdown_analyses_total{status}` | Drawdown analysis count. |
| `risk_recommendations_generated_total{severity}` | Generated advanced recommendation count. |
| `risk_analytics_duration_seconds_bucket{operation}` | Advanced analytics latency. |

No tenant, portfolio, user, trace, or correlation IDs are used as metric labels.

## Events

The risk service publishes analytics lifecycle events through its existing Kafka producer:

- `risk.stress_test.completed`
- `risk.scenario.completed`
- `risk.concentration.analyzed`
- `risk.drawdown.analyzed`

Schemas live under `schemas/events/risk/`.

## Demo

```bash
TOKEN=<jwt> ./scripts/demo-risk-analytics.sh
TOKEN=<jwt> ./scripts/demo-risk-analytics.sh --run
```

Example payloads live under `docs/examples/risk/`.

## Limitations

- Calculations use supplied/demo positions rather than live portfolio joins.
- GET portfolio-specific endpoints currently return deterministic demo data for future integration.
- Historical drawdown uses caller-supplied values; no warehouse integration is required.
- The formulas are interview-friendly approximations, not production market-risk models.
