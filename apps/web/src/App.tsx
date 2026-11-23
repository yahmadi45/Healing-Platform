import { useCallback, useEffect, useState } from 'react';
import {
  getIncidents,
  getServices,
  getStats,
  runServiceAction,
  type Incident,
  type Service,
  type ServiceAction,
  type Stats,
} from './api/client';
import { IncidentTimeline } from './components/IncidentTimeline';
import { ServicesPanel } from './components/ServicesPanel';
import { Sidebar } from './components/Sidebar';
import { SummaryCards } from './components/SummaryCards';

const REFRESH_INTERVAL_MS = 15_000;

export default function App() {
  const [services, setServices] = useState<Service[] | null>(null);
  const [incidents, setIncidents] = useState<Incident[] | null>(null);
  const [stats, setStats] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [busyService, setBusyService] = useState<string | null>(null);
  const [updatedAt, setUpdatedAt] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const [nextServices, nextIncidents, nextStats] = await Promise.all([
        getServices(),
        getIncidents(),
        getStats(),
      ]);
      setServices(nextServices);
      setIncidents(nextIncidents);
      setStats(nextStats);
      setError(null);
      setUpdatedAt(new Date().toLocaleTimeString());
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Unable to load dashboard data.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
    const timer = window.setInterval(() => void refresh(), REFRESH_INTERVAL_MS);
    return () => window.clearInterval(timer);
  }, [refresh]);

  const handleAction = async (service: Service, action: ServiceAction) => {
    setBusyService(service.name);
    setNotice(null);
    try {
      const result = await runServiceAction(service.name, action);
      setNotice(result.message);
      await refresh();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : 'Unable to complete simulated action.');
    } finally {
      setBusyService(null);
    }
  };

  return (
    <div className="app-shell">
      <Sidebar openIncidents={stats?.open_incidents} />
      <main className="main" id="overview">
        <header className="topbar">
          <div><span className="crumb">Workspace /</span> Operations</div>
          <div className="top-right">
            <span className="live"><i /> LIVE</span>
            <button className="icon-button" type="button" aria-label="Refresh dashboard" onClick={() => void refresh()}>↻</button>
            <div className="avatar" aria-label="SentinelOps">SO</div>
          </div>
        </header>

        <section className="page-heading">
          <div>
            <div className="eyebrow">INFRASTRUCTURE CONTROL PLANE</div>
            <h1>Operations overview</h1>
            <p>Monitor demo services, review incidents, and test policy-controlled recovery.</p>
          </div>
          <div className="heading-actions">
            <span className="updated-at">{updatedAt ? `Updated ${updatedAt}` : 'Waiting for API'}</span>
            <button className="button secondary" type="button" onClick={() => void refresh()}>Refresh data</button>
          </div>
        </section>

        <div className="simulation-banner">
          <span className="simulation-mark" aria-hidden="true">!</span>
          <span><strong>DEMO MODE</strong> Actions change SentinelOps in-memory state only. They never stop or restart real containers.</span>
        </div>
        {error && (
          <div className="notice error" role="alert">
            <span>{error}</span>
            <button type="button" onClick={() => void refresh()}>Retry</button>
          </div>
        )}
        {notice && <div className="notice" role="status">{notice}</div>}

        <SummaryCards stats={stats} />
        <ServicesPanel
          services={services}
          loading={loading}
          busyService={busyService}
          onAction={(service, action) => void handleAction(service, action)}
        />
        <IncidentTimeline incidents={incidents} loading={loading} />

        <footer>
          SentinelOps MVP <span>•</span> Simulated actions only <span>•</span>
          <a href="/metrics" target="_blank" rel="noreferrer">Metrics endpoint</a>
        </footer>
      </main>
    </div>
  );
}
