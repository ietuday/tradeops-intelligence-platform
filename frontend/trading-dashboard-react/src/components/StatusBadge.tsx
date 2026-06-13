import { Status } from '../types';

export function StatusBadge({ status }: { status?: string }) {
  const normalized = (status || 'UNKNOWN').toUpperCase() as Status;
  return <span className={`badge badge-${normalized.toLowerCase()}`}>{normalized}</span>;
}
