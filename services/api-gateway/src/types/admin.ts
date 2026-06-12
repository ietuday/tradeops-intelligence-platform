export type AdminStatus = 'HEALTHY' | 'DEGRADED' | 'UNHEALTHY';

export interface ServiceRegistryEntry {
  name: string;
  type: 'node' | 'go' | 'python';
  category: string;
  baseUrl: string;
  healthPath: string;
  readyPath: string;
  metricsPath: string;
  ownsData: boolean;
  critical: boolean;
  producesTopics: string[];
  consumesTopics: string[];
}

export interface TopicCatalogEntry {
  topic: string;
  producer: string;
  consumers: string[];
  schema: string;
  version: string;
  description: string;
}

export interface DlqEntry {
  topic: string;
  owner: string;
  description: string;
  replayScript: string;
  runbook: string;
  status: 'STATIC_CATALOG';
}

export interface AdminContext {
  tenantId: string;
  userId?: string;
  roles: string[];
  authorization: string;
  correlationId?: string;
}

export interface DownstreamResult<T> {
  status: 'OK' | 'DEGRADED';
  data: T;
  error?: string;
}
