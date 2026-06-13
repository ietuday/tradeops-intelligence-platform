export type Status = 'HEALTHY' | 'DEGRADED' | 'UNHEALTHY' | 'UNKNOWN';

export interface DashboardContext {
  token: string;
  tenantId: string;
}

export interface ApiRequestOptions extends DashboardContext {
  signal?: AbortSignal;
}

export interface ServiceHealth {
  name: string;
  status: Status;
  healthUrl: string;
  latencyMs: number;
  error: string | null;
}

export interface HealthSummary {
  status: Status;
  checkedAt: string;
  services: ServiceHealth[];
}

export interface SummaryResponse {
  tenantId?: string;
  status?: string;
  summary?: Record<string, number>;
  [key: string]: unknown;
}

export interface RiskScenario {
  name: string;
  marketShockPercent?: number;
  volatilityMultiplier?: number;
  symbolShocks?: Record<string, number>;
  sectorShocks?: Record<string, number>;
  liquidityHaircutPercent?: number;
}

export interface RiskPosition {
  symbol: string;
  quantity: number;
  averagePrice: number;
  currentPrice: number;
  sector: string;
  assetClass: string;
}

export interface StressTestRequest {
  portfolioId: string;
  positions: RiskPosition[];
  scenarios: RiskScenario[];
}

export interface StressTestResult {
  portfolioId: string;
  baselineValue: number;
  stressedValue: number;
  pnlImpact: number;
  pnlImpactPercent: number;
  worstScenario: string | null;
  scenarioResults: Array<{ scenarioName: string; riskLevel: string; pnlImpactPercent: number }>;
  recommendations: Array<{ code: string; severity: string; message: string; suggestedAction: string }>;
}

export interface RealtimeEvent {
  type?: string;
  topic?: string;
  stream?: string;
  correlationId?: string;
  tenantId?: string;
  timestamp?: string;
  payload?: unknown;
  [key: string]: unknown;
}
