import { useEffect, useState } from 'react';
import { api } from '../api/client';
import { MetricCard } from '../components/MetricCard';
import { StatusBadge } from '../components/StatusBadge';
import { DashboardContext, HealthSummary, SummaryResponse } from '../types';

export function DashboardPage(context: DashboardContext) {
  const [health, setHealth] = useState<HealthSummary | null>(null);
  const [alerts, setAlerts] = useState<SummaryResponse | null>(null);
  const [notifications, setNotifications] = useState<SummaryResponse | null>(null);
  const [dlqs, setDlqs] = useState<unknown[]>([]);
  const [rules, setRules] = useState<SummaryResponse | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    setError('');
    Promise.all([
      api.getHealthSummary(context),
      api.getAlertsSummary(context),
      api.getNotificationsSummary(context),
      api.getDlqSummary(context),
      api.getRuleConfigSummary(context)
    ])
      .then(([healthResult, alertResult, notificationResult, dlqResult, ruleResult]) => {
        setHealth(healthResult);
        setAlerts(alertResult);
        setNotifications(notificationResult);
        setDlqs(dlqResult.dlqs || []);
        setRules(ruleResult);
      })
      .catch((err: Error) => setError(err.message));
  }, [context.token, context.tenantId]);

  const alertSummary = alerts?.summary || {};
  const notificationSummary = notifications?.summary || {};
  const ruleCounts = (rules?.rules as Record<string, number> | undefined) || {};

  return (
    <div className="page">
      <div className="page-title">
        <h1>Dashboard</h1>
        <p>Tenant: {context.tenantId}</p>
      </div>
      {error && <div className="error">Unable to load dashboard: {error}</div>}
      <section className="metric-grid">
        <MetricCard title="Platform" value={<StatusBadge status={health?.status} />} detail={health?.checkedAt || 'Waiting for admin API'} />
        <MetricCard title="Open Alerts" value={alertSummary.open ?? 0} detail={`${alertSummary.critical ?? 0} critical`} />
        <MetricCard title="Failed Notifications" value={notificationSummary.failed ?? 0} detail={`${notificationSummary.pendingRetries ?? 0} pending retries`} />
        <MetricCard title="DLQ Topics" value={dlqs.length} detail="Static operations catalog" />
        <MetricCard title="Enabled Rules" value={ruleCounts.enabled ?? 0} detail={`${ruleCounts.disabled ?? 0} disabled`} />
      </section>
      <section className="panel">
        <h2>Service Health</h2>
        <table>
          <thead><tr><th>Service</th><th>Status</th><th>Latency</th><th>Error</th></tr></thead>
          <tbody>
            {(health?.services || []).map((service) => (
              <tr key={service.name}>
                <td>{service.name}</td>
                <td><StatusBadge status={service.status} /></td>
                <td>{service.latencyMs} ms</td>
                <td>{service.error || '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
