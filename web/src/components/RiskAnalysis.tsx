export default function RiskAnalysis() {
  const metrics = [
    { label: 'Sharpe Ratio', value: '1.42', sub: 'Risk-adj. return', color: '#10b981' },
    { label: 'Sortino Ratio', value: '1.87', sub: 'Downside risk-adj.', color: '#10b981' },
    { label: 'Volatility', value: '8.8%', sub: 'Annualized', color: '#eab308' },
    { label: 'Max Drawdown', value: '-12.4%', sub: 'Peak to trough', color: '#ef4444' },
    { label: 'Top Concentration', value: '32%', sub: 'BTC/USDT', color: '#eab308' },
  ]

  return (
    <div className="card fade-in" style={{ animationDelay: '100ms', display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      <div style={{ padding: '12px 20px', borderBottom: '1px solid var(--border)' }}>
        <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-1)' }}>Risk Analysis</span>
      </div>

      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', padding: '8px 0' }}>
        {metrics.map((m, i) => (
          <div
            key={m.label}
            style={{
              display: 'flex', alignItems: 'center', justifyContent: 'space-between',
              padding: '12px 20px',
              borderBottom: i < metrics.length - 1 ? '1px solid rgba(255,255,255,0.025)' : 'none',
            }}
          >
            <div>
              <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-2)' }}>{m.label}</div>
              <div style={{ fontSize: 9, color: 'var(--text-3)', marginTop: 2 }}>{m.sub}</div>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <div style={{
                width: 4, height: 24, borderRadius: 2,
                background: m.color, opacity: 0.6,
              }} />
              <span className="mono" style={{ fontSize: 16, fontWeight: 800, color: m.color }}>
                {m.value}
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
