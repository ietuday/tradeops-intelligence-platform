import { config } from '../config';
import { ApiRequestOptions, HealthSummary, RiskScenario, StressTestRequest, StressTestResult, SummaryResponse } from '../types';

export class ApiError extends Error {
  constructor(message: string, readonly status?: number) {
    super(message);
    this.name = 'ApiError';
  }
}

export function correlationId(): string {
  return `dashboard-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export async function requestJson<T>(path: string, options: ApiRequestOptions, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set('accept', 'application/json');
  headers.set('x-tenant-id', options.tenantId || config.defaultTenantId);
  headers.set('x-correlation-id', correlationId());
  if (options.token) {
    headers.set('authorization', `Bearer ${options.token}`);
  }
  if (init.body && !headers.has('content-type')) {
    headers.set('content-type', 'application/json');
  }

  const response = await fetch(`${config.apiBaseUrl}${path}`, {
    ...init,
    headers,
    signal: options.signal
  });
  if (!response.ok) {
    throw new ApiError(`Request failed with HTTP ${response.status}`, response.status);
  }
  return response.json() as Promise<T>;
}

export const api = {
  getHealthSummary: (options: ApiRequestOptions) => requestJson<HealthSummary>('/api/admin/health-summary', options),
  getServices: (options: ApiRequestOptions) => requestJson<{ services: unknown[] }>('/api/admin/services', options),
  getTopics: (options: ApiRequestOptions) => requestJson<{ topics: unknown[] }>('/api/admin/topics', options),
  getDlqSummary: (options: ApiRequestOptions) => requestJson<{ dlqs: unknown[] }>('/api/admin/dlq-summary', options),
  getAlertsSummary: (options: ApiRequestOptions) => requestJson<SummaryResponse>('/api/admin/alerts-summary', options),
  getNotificationsSummary: (options: ApiRequestOptions) => requestJson<SummaryResponse>('/api/admin/notifications-summary', options),
  getRuleConfigSummary: (options: ApiRequestOptions) => requestJson<SummaryResponse>('/api/admin/rule-config-summary', options),
  getPlatformConfig: (options: ApiRequestOptions) => requestJson<Record<string, unknown>>('/api/admin/platform-config', options),
  getOpsChecklist: (options: ApiRequestOptions) => requestJson<{ checklists: unknown[] }>('/api/admin/ops-checklist', options),
  getRiskScenarios: (options: ApiRequestOptions) => requestJson<{ scenarios: RiskScenario[] }>('/api/risk/scenarios', options),
  runStressTest: (options: ApiRequestOptions, body: StressTestRequest = sampleStressTest()) => post<StressTestResult>('/api/risk/stress-test', options, body),
  runConcentrationAnalysis: (options: ApiRequestOptions) => post<Record<string, unknown>>('/api/risk/portfolio/concentration', options, sampleConcentration()),
  runDrawdownTrend: (options: ApiRequestOptions) => post<Record<string, unknown>>('/api/risk/portfolio/drawdown-trend', options, sampleDrawdown()),
  runVolatilityShock: (options: ApiRequestOptions) => post<Record<string, unknown>>('/api/risk/volatility-shock', options, sampleVolatilityShock())
};

function post<T>(path: string, options: ApiRequestOptions, body: unknown): Promise<T> {
  return requestJson<T>(path, options, {
    method: 'POST',
    body: JSON.stringify(body)
  });
}

export function sampleStressTest(): StressTestRequest {
  return {
    portfolioId: 'demo-portfolio-1',
    positions: [
      { symbol: 'AAPL', quantity: 10, averagePrice: 150, currentPrice: 180, sector: 'Technology', assetClass: 'EQUITY' },
      { symbol: 'JPM', quantity: 5, averagePrice: 140, currentPrice: 150, sector: 'Financials', assetClass: 'EQUITY' }
    ],
    scenarios: [
      { name: 'Market drops 10%', marketShockPercent: -10 },
      { name: 'Technology drops 15%', marketShockPercent: -5, sectorShocks: { Technology: -15 } }
    ]
  };
}

function sampleConcentration() {
  return {
    portfolioId: 'demo-portfolio-1',
    positions: [
      { symbol: 'AAPL', quantity: 10, averagePrice: 150, currentPrice: 180, sector: 'Technology', assetClass: 'EQUITY' },
      { symbol: 'MSFT', quantity: 8, averagePrice: 250, currentPrice: 300, sector: 'Technology', assetClass: 'EQUITY' },
      { symbol: 'JPM', quantity: 5, averagePrice: 140, currentPrice: 150, sector: 'Financials', assetClass: 'EQUITY' }
    ]
  };
}

function sampleDrawdown() {
  return {
    portfolioId: 'demo-portfolio-1',
    values: [{ value: 10000 }, { value: 11200 }, { value: 9800 }, { value: 9100 }, { value: 10400 }]
  };
}

function sampleVolatilityShock() {
  return {
    portfolioId: 'demo-portfolio-1',
    volatilityMultiplier: 2,
    baseRiskScore: 30,
    positions: [{ symbol: 'AAPL', quantity: 10, averagePrice: 150, currentPrice: 180, sector: 'Technology', assetClass: 'EQUITY' }]
  };
}
