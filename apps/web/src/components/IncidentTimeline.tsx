import type { Incident } from '../api/client';

interface IncidentTimelineProps {
  incidents: Incident[] | null;
  loading: boolean;
}

function formatTime(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

export function IncidentTimeline({ incidents, loading }: IncidentTimelineProps) {
  return (
    <section className="panel incidents-panel" id="incidents" aria-labelledby="incidents-title">
      <div className="panel-heading">
        <div>
          <h2 id="incidents-title">Incident timeline</h2>
          <p>Recent simulated alerts and policy decisions</p>
        </div>
        <span className="subtle">Latest first</span>
      </div>
      <div className="timeline">
        {loading && incidents === null ? (
          <div className="empty">Loading incidents…</div>
        ) : incidents === null ? (
          <div className="empty">Incidents are unavailable.</div>
        ) : incidents.length === 0 ? (
          <div className="empty">No incidents. Services are operating normally.</div>
        ) : incidents.map((incident) => {
          const resolved = incident.status === 'resolved';
          return (
            <article className="incident" key={incident.id}>
              <span className={`incident-mark ${resolved ? 'success' : ''}`} aria-hidden="true">
                {resolved ? '✓' : '!'}
              </span>
              <div>
                <div className="incident-title">{incident.title}</div>
                <div className="incident-detail">{incident.service} · {incident.detail}</div>
              </div>
              <time className="incident-time" dateTime={incident.created_at}>
                {formatTime(incident.created_at)}
              </time>
            </article>
          );
        })}
      </div>
    </section>
  );
}
