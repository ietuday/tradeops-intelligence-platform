import { config } from '../config';

export function ObservabilityPage() {
  return (
    <div className="page">
      <div className="page-title">
        <h1>Observability</h1>
        <p>Jump into the local observability tools and correlate dashboard events.</p>
      </div>
      <section className="metric-grid">
        <a className="link-card" href={config.grafanaUrl} target="_blank" rel="noreferrer"><strong>Grafana</strong><span>{config.grafanaUrl}</span></a>
        <a className="link-card" href={config.jaegerUrl} target="_blank" rel="noreferrer"><strong>Jaeger</strong><span>{config.jaegerUrl}</span></a>
        <a className="link-card" href={config.prometheusUrl} target="_blank" rel="noreferrer"><strong>Prometheus</strong><span>{config.prometheusUrl}</span></a>
      </section>
      <section className="panel">
        <h2>Correlation Notes</h2>
        <p>Use `correlationId` from WebSocket events and API responses to search logs and audit records. Use trace IDs in Jaeger for span timing and request/event flow inspection.</p>
        <p>Docs: `docs/tracing/opentelemetry.md`, `docs/observability/runbook.md`, `docs/realtime/websocket-streaming.md`.</p>
      </section>
    </div>
  );
}
