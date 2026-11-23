import type { Service, ServiceAction } from '../api/client';

interface ServicesPanelProps {
  services: Service[] | null;
  loading: boolean;
  busyService: string | null;
  onAction: (service: Service, action: ServiceAction) => void;
}

function formatTime(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

export function ServicesPanel({
  services,
  loading,
  busyService,
  onAction,
}: ServicesPanelProps) {
  return (
    <section className="panel" id="services" aria-labelledby="services-title">
      <div className="panel-heading">
        <div>
          <h2 id="services-title">Services</h2>
          <p>Demo states only; no host containers are controlled.</p>
        </div>
      </div>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th scope="col">SERVICE</th>
              <th scope="col">ENVIRONMENT</th>
              <th scope="col">STATUS</th>
              <th scope="col">LAST CHECK</th>
              <th scope="col">SIMULATED ACTIONS</th>
            </tr>
          </thead>
          <tbody>
            {loading && services === null ? (
              <tr><td colSpan={5} className="empty">Loading services…</td></tr>
            ) : services === null ? (
              <tr><td colSpan={5} className="empty">Services are unavailable.</td></tr>
            ) : services.length === 0 ? (
              <tr><td colSpan={5} className="empty">No services registered.</td></tr>
            ) : services.map((service) => {
              const action: ServiceAction = service.status === 'healthy' ? 'fail' : 'recover';
              const actionLabel = action === 'fail' ? 'Simulate failure' : 'Simulate recovery';
              const isBusy = busyService === service.name;
              return (
                <tr key={service.name}>
                  <td>
                    <span className="service-name">
                      <span className="service-icon" aria-hidden="true">{service.icon}</span>
                      {service.name}
                    </span>
                  </td>
                  <td>{service.environment}</td>
                  <td><span className={`badge ${service.status}`}>{service.status}</span></td>
                  <td><time dateTime={service.last_check}>{formatTime(service.last_check)}</time></td>
                  <td>
                    <button
                      className={`small-button ${action === 'fail' ? 'warn' : ''}`}
                      type="button"
                      disabled={busyService !== null}
                      aria-label={`${actionLabel} for ${service.name}`}
                      onClick={() => onAction(service, action)}
                    >
                      {isBusy ? 'Updating…' : actionLabel}
                    </button>
                    <span className="action-note">simulation only</span>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </section>
  );
}
