import { useEffect, useState } from 'react';
import { api } from '../api/client';
import { StatusBadge } from '../components/StatusBadge';
import { DashboardContext, HealthSummary } from '../types';

export function AdminOpsPage(context: DashboardContext) {
  const [health, setHealth] = useState<HealthSummary | null>(null);
  const [services, setServices] = useState<unknown[]>([]);
  const [topics, setTopics] = useState<unknown[]>([]);
  const [dlqs, setDlqs] = useState<unknown[]>([]);
  const [platform, setPlatform] = useState<Record<string, unknown> | null>(null);
  const [checklists, setChecklists] = useState<unknown[]>([]);
  const [error, setError] = useState('');

  useEffect(() => {
    Promise.all([
      api.getHealthSummary(context),
      api.getServices(context),
      api.getTopics(context),
      api.getDlqSummary(context),
      api.getPlatformConfig(context),
      api.getOpsChecklist(context)
    ])
      .then(([healthResult, serviceResult, topicResult, dlqResult, platformResult, checklistResult]) => {
        setHealth(healthResult);
        setServices(serviceResult.services || []);
        setTopics(topicResult.topics || []);
        setDlqs(dlqResult.dlqs || []);
        setPlatform(platformResult);
        setChecklists(checklistResult.checklists || []);
      })
      .catch((err: Error) => setError(err.message));
  }, [context.token, context.tenantId]);

  return (
    <div className="page">
      <div className="page-title">
        <h1>Admin Ops</h1>
        <p>Read-only operations summaries from `/api/admin`.</p>
      </div>
      {error && <div className="error">{error}</div>}
      <section className="panel">
        <h2>Health Summary <StatusBadge status={health?.status} /></h2>
        <table>
          <thead><tr><th>Service</th><th>Status</th><th>Health URL</th></tr></thead>
          <tbody>{(health?.services || []).map((service) => <tr key={service.name}><td>{service.name}</td><td><StatusBadge status={service.status} /></td><td>{service.healthUrl}</td></tr>)}</tbody>
        </table>
      </section>
      <DataTable title="Services" rows={services} keys={['name', 'type', 'category', 'baseUrl']} />
      <DataTable title="Topics" rows={topics} keys={['topic', 'producer', 'schema']} />
      <DataTable title="DLQs" rows={dlqs} keys={['topic', 'owner', 'runbook']} />
      <DataTable title="Ops Checklist" rows={checklists} keys={['name', 'doc']} />
      <section className="panel">
        <h2>Platform Config</h2>
        <pre>{JSON.stringify(platform, null, 2)}</pre>
      </section>
    </div>
  );
}

function DataTable({ title, rows, keys }: { title: string; rows: unknown[]; keys: string[] }) {
  return (
    <section className="panel">
      <h2>{title}</h2>
      <table>
        <thead><tr>{keys.map((key) => <th key={key}>{key}</th>)}</tr></thead>
        <tbody>{rows.slice(0, 12).map((row, index) => {
          const item = row as Record<string, unknown>;
          return <tr key={index}>{keys.map((key) => <td key={key}>{String(item[key] ?? '-')}</td>)}</tr>;
        })}</tbody>
      </table>
    </section>
  );
}
