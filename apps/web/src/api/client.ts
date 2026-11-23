export interface Service {
  name: string;
  environment: string;
  status: string;
  icon: string;
  last_check: string;
  recoveries: number;
}

export interface Incident {
  id: string;
  service: string;
  title: string;
  detail: string;
  status: string;
  created_at: string;
}

export interface Stats {
  services: number;
  healthy: number;
  open_incidents: number;
  recoveries: number;
}

export type ServiceAction = 'fail' | 'recover';

interface ActionResponse {
  message: string;
}

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, init);
  const text = await response.text();
  let body: unknown;

  if (text) {
    try {
      body = JSON.parse(text) as unknown;
    } catch {
      if (response.ok) {
        throw new Error('The API returned invalid JSON.');
      }
    }
  }

  if (!response.ok) {
    const errorMessage =
      typeof body === 'object' &&
      body !== null &&
      'error' in body &&
      typeof body.error === 'string'
        ? body.error
        : `Request failed (${response.status}).`;
    throw new Error(errorMessage);
  }

  return body as T;
}

export async function getServices(): Promise<Service[]> {
  return request<Service[]>('/api/services');
}

export async function getIncidents(): Promise<Incident[]> {
  return (await request<Incident[] | null>('/api/incidents')) ?? [];
}

export function getStats(): Promise<Stats> {
  return request<Stats>('/api/stats');
}

export function runServiceAction(
  name: string,
  action: ServiceAction,
): Promise<ActionResponse> {
  return request<ActionResponse>(
    `/api/services/${encodeURIComponent(name)}/${action}`,
    { method: 'POST' },
  );
}
