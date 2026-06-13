export const config = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
  wsBaseUrl: import.meta.env.VITE_WS_BASE_URL || 'ws://localhost:8080',
  grafanaUrl: import.meta.env.VITE_GRAFANA_URL || 'http://localhost:3000',
  jaegerUrl: import.meta.env.VITE_JAEGER_URL || 'http://localhost:16686',
  prometheusUrl: import.meta.env.VITE_PROMETHEUS_URL || 'http://localhost:9090',
  defaultTenantId: import.meta.env.VITE_DEFAULT_TENANT_ID || 'default-tenant'
};
