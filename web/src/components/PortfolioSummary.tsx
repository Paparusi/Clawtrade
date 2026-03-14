const stats = [
  { label: 'Portfolio Value', value: '$10,245.80', change: '+$245.80', pct: '+2.45%', up: true },
  { label: 'Unrealized P&L', value: '+$747.00', change: '6 positions', pct: '+7.29%', up: true },
  { label: "Today's P&L", value: '-$45.20', change: 'Since 00:00 UTC', pct: '-0.44%', up: false },
  { label: 'Win Rate', value: '68.5%', change: '94W / 43L', pct: '1.85 R:R', up: true },
]

export default function PortfolioSummary() {
  return (
    <div className="card fade-in" style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)' }}>
      {stats.map((s, i) => (
        <div key={s.label} style={{
          padding: '18px 24px',
          borderRight: i < stats.length - 1 ? '1px solid var(--border)' : 'none',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10 }}>
            <span style={{ fontSize: 11, color: 'var(--text-3)', fontWeight: 500 }}>{s.label}</span>
            <span className="mono" style={{
              fontSize: 10, fontWeight: 600, padding: '2px 6px', borderRadius: 4,
              color: s.up ? '#10b981' : '#ef4444',
              background: s.up ? 'rgba(16,185,129,0.08)' : 'rgba(239,68,68,0.08)',
            }}>{s.pct}</span>
          </div>
          <div className="mono" style={{
            fontSize: 22, fontWeight: 800, letterSpacing: '-0.02em', lineHeight: 1,
            color: s.label.includes('P&L') ? (s.up ? '#10b981' : '#ef4444') : 'var(--text-1)',
          }}>{s.value}</div>
          <div style={{ fontSize: 10, color: 'var(--text-3)', marginTop: 6 }}>{s.change}</div>
        </div>
      ))}
    </div>
  )
}
