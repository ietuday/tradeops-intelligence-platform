import { RealtimeEventPanel } from '../components/RealtimeEventPanel';
import { DashboardContext } from '../types';

export function RealtimePage(context: DashboardContext) {
  return (
    <div className="page">
      <div className="page-title">
        <h1>Realtime</h1>
        <p>Connect to API Gateway WebSocket streams.</p>
      </div>
      <RealtimeEventPanel {...context} />
    </div>
  );
}
