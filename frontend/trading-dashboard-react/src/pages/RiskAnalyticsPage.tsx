import { useEffect, useState } from 'react';
import { api, sampleStressTest } from '../api/client';
import { MetricCard } from '../components/MetricCard';
import { DashboardContext, RiskScenario, StressTestRequest, StressTestResult } from '../types';

export function RiskAnalyticsPage(context: DashboardContext) {
  const [scenarios, setScenarios] = useState<RiskScenario[]>([]);
  const [stressJson, setStressJson] = useState(JSON.stringify(sampleStressTest(), null, 2));
  const [stressResult, setStressResult] = useState<StressTestResult | null>(null);
  const [concentration, setConcentration] = useState<Record<string, unknown> | null>(null);
  const [drawdown, setDrawdown] = useState<Record<string, unknown> | null>(null);
  const [volatility, setVolatility] = useState<Record<string, unknown> | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    api.getRiskScenarios(context).then((result) => setScenarios(result.scenarios || [])).catch((err: Error) => setError(err.message));
  }, [context.token, context.tenantId]);

  const runStress = () => {
    setError('');
    try {
      api.runStressTest(context, JSON.parse(stressJson) as StressTestRequest).then(setStressResult).catch((err: Error) => setError(err.message));
    } catch {
      setError('Stress test JSON is invalid.');
    }
  };

  return (
    <div className="page">
      <div className="page-title">
        <h1>Risk Analytics</h1>
        <p>Run deterministic stress testing demos against supplied positions.</p>
      </div>
      {error && <div className="error">{error}</div>}
      <section className="panel">
        <h2>Built-In Scenarios</h2>
        <div className="pill-row">
          {scenarios.map((scenario) => <span key={scenario.name} className="pill">{scenario.name}</span>)}
          {scenarios.length === 0 && <span className="empty">No scenarios loaded.</span>}
        </div>
      </section>
      <section className="panel split-panel">
        <div>
          <h2>Stress Test</h2>
          <textarea value={stressJson} onChange={(event) => setStressJson(event.target.value)} rows={18} />
          <button onClick={runStress}>Run Stress Test</button>
        </div>
        <div>
          <h2>Result</h2>
          {stressResult ? (
            <>
              <div className="metric-grid compact">
                <MetricCard title="Baseline" value={money(stressResult.baselineValue)} />
                <MetricCard title="Stressed" value={money(stressResult.stressedValue)} />
                <MetricCard title="PnL" value={`${stressResult.pnlImpactPercent.toFixed(2)}%`} />
                <MetricCard title="Worst Scenario" value={stressResult.worstScenario || 'n/a'} />
              </div>
              <h3>Recommendations</h3>
              <ul>
                {stressResult.recommendations.map((item) => <li key={item.code}>{item.message} {item.suggestedAction}</li>)}
              </ul>
            </>
          ) : <p className="empty">Run the sample to see results.</p>}
        </div>
      </section>
      <section className="metric-grid">
        <ActionCard title="Concentration" onRun={() => api.runConcentrationAnalysis(context).then(setConcentration)} result={concentration} />
        <ActionCard title="Drawdown Trend" onRun={() => api.runDrawdownTrend(context).then(setDrawdown)} result={drawdown} />
        <ActionCard title="Volatility Shock" onRun={() => api.runVolatilityShock(context).then(setVolatility)} result={volatility} />
      </section>
    </div>
  );
}

function ActionCard({ title, onRun, result }: { title: string; onRun: () => void; result: Record<string, unknown> | null }) {
  return (
    <article className="panel mini-panel">
      <h2>{title}</h2>
      <button onClick={onRun}>Run Sample</button>
      {result && <pre>{JSON.stringify(result, null, 2).slice(0, 900)}</pre>}
    </article>
  );
}

function money(value: number) {
  return `$${value.toLocaleString(undefined, { maximumFractionDigits: 2 })}`;
}
