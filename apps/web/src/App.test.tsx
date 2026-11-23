import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import {
  getIncidents,
  getServices,
  getStats,
  runServiceAction,
  type Service,
} from './api/client';

vi.mock('./api/client', () => ({
  getIncidents: vi.fn(),
  getServices: vi.fn(),
  getStats: vi.fn(),
  runServiceAction: vi.fn(),
}));

const healthyService: Service = {
  name: 'api',
  environment: 'development',
  status: 'healthy',
  icon: '⌘',
  last_check: '2026-01-01T00:00:00Z',
  recoveries: 0,
};

beforeEach(() => {
  vi.mocked(getServices).mockResolvedValue([healthyService]);
  vi.mocked(getIncidents).mockResolvedValue([]);
  vi.mocked(getStats).mockResolvedValue({
    services: 1,
    healthy: 1,
    open_incidents: 0,
    recoveries: 0,
  });
  vi.mocked(runServiceAction).mockResolvedValue({ message: 'Simulated failure recorded.' });
});

describe('operations dashboard', () => {
  it('renders service and summary data and refreshes after a simulated action', async () => {
    const user = userEvent.setup();
    const unhealthyService = { ...healthyService, status: 'unhealthy' };
    vi.mocked(getServices)
      .mockResolvedValueOnce([healthyService])
      .mockResolvedValueOnce([unhealthyService]);
    vi.mocked(getStats)
      .mockResolvedValueOnce({ services: 1, healthy: 1, open_incidents: 0, recoveries: 0 })
      .mockResolvedValueOnce({ services: 1, healthy: 0, open_incidents: 1, recoveries: 0 });

    render(<App />);

    expect(await screen.findByText('development')).toBeInTheDocument();
    expect(screen.getByText('Services monitored').parentElement).toHaveTextContent('1');
    await user.click(screen.getByRole('button', { name: 'Simulate failure for api' }));

    await waitFor(() => expect(runServiceAction).toHaveBeenCalledWith('api', 'fail'));
    expect(await screen.findByRole('button', { name: 'Simulate recovery for api' })).toBeInTheDocument();
    expect(screen.getByText('Simulated failure recorded.')).toBeInTheDocument();
    expect(getServices).toHaveBeenCalledTimes(2);
  });

  it('shows an API error and a retry control', async () => {
    vi.mocked(getServices).mockRejectedValueOnce(new Error('API unavailable'));

    render(<App />);

    expect(await screen.findByRole('alert')).toHaveTextContent('API unavailable');
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
  });
});
