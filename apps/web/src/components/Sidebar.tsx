interface SidebarProps {
  openIncidents: number | undefined;
}

export function Sidebar({ openIncidents }: SidebarProps) {
  return (
    <aside className="sidebar">
      <a className="brand" href="#overview" aria-label="SentinelOps overview">
        <span className="brand-mark">S</span>
        <span className="brand-name">
          Sentinel<span className="brand-muted">Ops</span>
          <small>DEVNET LAB</small>
        </span>
      </a>
      <div className="nav-label">WORKSPACE</div>
      <nav aria-label="Workspace">
        <a className="nav-item active" href="#overview" aria-current="page">
          <span aria-hidden="true">◫</span> Overview
        </a>
        <a className="nav-item" href="#services">
          <span aria-hidden="true">◇</span> Services
        </a>
        <a className="nav-item" href="#incidents">
          <span aria-hidden="true">◎</span> Incidents
          <span className="nav-count">{openIncidents ?? '—'}</span>
        </a>
      </nav>
      <div className="sidebar-bottom">
        <span className="pulse-dot" /> Demo environment <span className="version">MVP</span>
      </div>
    </aside>
  );
}
