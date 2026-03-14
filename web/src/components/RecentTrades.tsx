const trades = [
  { sym: 'BTC/USDT', exchange: 'Binance', side: 'Buy', qty: '0.05 BTC', price: '68,500', time: '14:32', pnl: null },
  { sym: 'EUR/USD', exchange: 'MT5', side: 'Sell', qty: '0.5 lot', price: '1.0892', time: '14:15', pnl: null },
  { sym: 'SOL/USDT', exchange: 'Hyperliquid', side: 'Sell', qty: '15 SOL', price: '178.5', time: '13:15', pnl: 93 },
  { sym: 'AAPL', exchange: 'IBKR', side: 'Buy', qty: '25 shares', price: '195.40', time: '12:30', pnl: null },
  { sym: 'BNB/USDT', exchange: 'Binance', side: 'Sell', qty: '2 BNB', price: '625.4', time: '10:22', pnl: -15 },
]

export default function RecentTrades() {
  return (
    <div className="card fade-in" style={{ flex: 1, display: 'flex', flexDirection: 'column', animationDelay: '300ms' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 16px', borderBottom: '1px solid var(--border)' }}>
        <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-1)' }}>Activity</span>
        <span style={{ fontSize: 10, color: 'var(--text-3)' }}>Today</span>
      </div>
      <div style={{ flex: 1 }}>
        {trades.map((t, i) => (
          <div
            key={i}
            style={{
              display: 'flex', alignItems: 'center', justifyContent: 'space-between',
              padding: '8px 16px',
              borderBottom: '1px solid rgba(255,255,255,0.02)',
              cursor: 'pointer', transition: 'background 0.15s',
            }}
            onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255,255,255,0.015)'}
            onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{
                fontSize: 9, fontWeight: 600, padding: '1px 5px', borderRadius: 3,
                color: t.side === 'Buy' ? '#10b981' : '#ef4444',
                background: t.side === 'Buy' ? 'rgba(16,185,129,0.08)' : 'rgba(239,68,68,0.08)',
              }}>{t.side}</span>
              <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-1)' }}>{t.sym}</span>
              <span style={{ fontSize: 9, color: 'var(--text-3)' }}>{t.exchange}</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              {t.pnl !== null ? (
                <span className="mono" style={{ fontSize: 11, fontWeight: 600, color: t.pnl >= 0 ? '#10b981' : '#ef4444' }}>
                  {t.pnl >= 0 ? '+' : ''}{t.pnl}
                </span>
              ) : (
                <span style={{ fontSize: 9, fontWeight: 500, color: '#6366f1', background: 'rgba(99,102,241,0.08)', padding: '1px 5px', borderRadius: 3 }}>Open</span>
              )}
              <span className="mono" style={{ fontSize: 9, color: 'var(--text-3)' }}>{t.time}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
