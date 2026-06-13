import { useState } from 'react';
import { AuthTokenPanel } from './components/AuthTokenPanel';
import { config } from './config';
import { AdminOpsPage } from './pages/AdminOpsPage';
import { DashboardPage } from './pages/DashboardPage';
import { ObservabilityPage } from './pages/ObservabilityPage';
import { RealtimePage } from './pages/RealtimePage';
import { RiskAnalyticsPage } from './pages/RiskAnalyticsPage';

type Page = 'dashboard' | 'realtime' | 'risk' | 'admin' | 'observability';

const nav: Array<{ page: Page; label: string }> = [
  { page: 'dashboard', label: 'Dashboard' },
  { page: 'realtime', label: 'Realtime' },
  { page: 'risk', label: 'Risk Analytics' },
  { page: 'admin', label: 'Admin Ops' },
  { page: 'observability', label: 'Observability' }
];

export function App() {
  const [page, setPage] = useState<Page>('dashboard');
  const [token, setToken] = useState(() => localStorage.getItem('tradeops.token') || '');
  const [tenantId, setTenantId] = useState(() => localStorage.getItem('tradeops.tenantId') || config.defaultTenantId);
  const context = { token, tenantId };

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <strong>TradeOps</strong>
          <span>Real-Time Dashboard</span>
        </div>
        <nav>
          {nav.map((item) => (
            <button key={item.page} className={page === item.page ? 'active' : ''} onClick={() => setPage(item.page)}>{item.label}</button>
          ))}
        </nav>
        <AuthTokenPanel token={token} tenantId={tenantId} onTokenChange={setToken} onTenantChange={setTenantId} />
      </aside>
      <main>
        {page === 'dashboard' && <DashboardPage {...context} />}
        {page === 'realtime' && <RealtimePage {...context} />}
        {page === 'risk' && <RiskAnalyticsPage {...context} />}
        {page === 'admin' && <AdminOpsPage {...context} />}
        {page === 'observability' && <ObservabilityPage />}
      </main>
    </div>
  );
}
