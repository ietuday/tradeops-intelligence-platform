import { afterEach, describe, expect, it, vi } from 'vitest';
import { requestJson } from './client';

describe('requestJson', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('adds auth, tenant, and correlation headers', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ ok: true }), {
      status: 200,
      headers: { 'content-type': 'application/json' }
    }));
    vi.stubGlobal('fetch', fetchMock);

    await requestJson('/api/admin/services', { token: 'token-1', tenantId: 'tenant-a' });

    const [, init] = fetchMock.mock.calls[0];
    const headers = init.headers as Headers;
    expect(headers.get('authorization')).toBe('Bearer token-1');
    expect(headers.get('x-tenant-id')).toBe('tenant-a');
    expect(headers.get('x-correlation-id')).toMatch(/^dashboard-/);
  });
});
