import type { Stats } from '../api/client';

interface SummaryCardsProps {
  stats: Stats | null;
}

const cards = [
  { key: 'services', label: 'Services monitored', note: 'Registered demo services', icon: '◇', tone: 'blue' },
  { key: 'healthy', label: 'Healthy services', note: 'Operational status', icon: '✓', tone: 'green' },
  { key: 'open_incidents', label: 'Open incidents', note: 'Requires review or recovery', icon: '!', tone: 'amber' },
  { key: 'recoveries', label: 'Recoveries', note: 'Actions logged this session', icon: '↻', tone: 'violet' },
] as const;

export function SummaryCards({ stats }: SummaryCardsProps) {
  return (
    <section className="stats-grid" aria-label="System summary">
      {cards.map((card) => (
        <article className="stat-card" key={card.key}>
          <div className="stat-label">
            {card.label}
            <span className={`stat-icon ${card.tone}`} aria-hidden="true">{card.icon}</span>
          </div>
          <div className="stat-value">{stats?.[card.key] ?? '—'}</div>
          <div className="stat-foot">{card.note}</div>
        </article>
      ))}
    </section>
  );
}
