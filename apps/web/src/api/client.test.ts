import { afterEach, describe, expect, it, vi } from 'vitest';
import { getIncidents, getServices, getStats, runServiceAction } from './client';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('API client', () => {
  it('uses the existing API routes and normalizes an empty incident response', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response('[]', { status: 200 }))
      .mockResolvedValueOnce(new Response('null', { status: 200 }))
      .mockResolvedValueOnce(new Response('{"services":0,"healthy":0,"recoveries":0,"open_incidents":0}', { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);

    await expect(getServices()).resolves.toEqual([]);
    await expect(getIncidents()).resolves.toEqual([]);
    await expect(getStats()).resolves.toMatchObject({ services: 0, open_incidents: 0 });
    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      '/api/services',
      '/api/incidents',
      '/api/stats',
    ]);
  });

  it('encodes service names and surfaces API errors', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response('{"message":"simulated"}', { status: 200 }))
      .mockResolvedValueOnce(new Response('{"error":"actions disabled"}', { status: 403 }));
    vi.stubGlobal('fetch', fetchMock);

    await expect(runServiceAction('api/test', 'fail')).resolves.toEqual({ message: 'simulated' });
    expect(fetchMock.mock.calls[0]).toEqual([
      '/api/services/api%2Ftest/fail',
      { method: 'POST' },
    ]);
    await expect(getStats()).rejects.toThrow('actions disabled');
  });
});
