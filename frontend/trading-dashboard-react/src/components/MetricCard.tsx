import { ReactNode } from 'react';

export function MetricCard({ title, value, detail }: { title: string; value: ReactNode; detail?: ReactNode }) {
  return (
    <article className="metric-card">
      <span>{title}</span>
      <strong>{value}</strong>
      {detail && <small>{detail}</small>}
    </article>
  );
}
