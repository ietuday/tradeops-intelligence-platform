import { recordProxyUpstreamError, recordProxyUpstreamTimeout } from '../observability/metrics';

const DEFAULT_PROXY_TIMEOUT_MS = 10_000;

export class ProxyError extends Error {
  service: string;
  status: number;
  code: string;

  constructor(service: string, status: number, code: string, message: string) {
    super(message);
    this.name = 'ProxyError';
    this.service = service;
    this.status = status;
    this.code = code;
  }
}

export function isProxyError(error: unknown): error is ProxyError {
  return error instanceof ProxyError;
}

export function proxyTimeoutMs(): number {
  const configured = Number(process.env.PROXY_TIMEOUT_MS || DEFAULT_PROXY_TIMEOUT_MS);
  if (!Number.isFinite(configured) || configured <= 0) {
    return DEFAULT_PROXY_TIMEOUT_MS;
  }
  return configured;
}

export async function fetchUpstream(service: string, url: string, init: RequestInit): Promise<Response> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), proxyTimeoutMs());

  try {
    return await fetch(url, {
      ...init,
      signal: controller.signal
    });
  } catch (error) {
    if (isAbortError(error)) {
      recordProxyUpstreamTimeout(service);
      recordProxyUpstreamError(service, 504);
      throw new ProxyError(service, 504, 'UPSTREAM_TIMEOUT', `${service} did not respond before the proxy timeout.`);
    }

    recordProxyUpstreamError(service, 502);
    throw new ProxyError(service, 502, 'UPSTREAM_UNAVAILABLE', `${service} is unavailable.`);
  } finally {
    clearTimeout(timeout);
  }
}

function isAbortError(error: unknown): boolean {
  return error instanceof Error && error.name === 'AbortError';
}
