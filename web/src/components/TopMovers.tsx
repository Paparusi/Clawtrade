const movers = [
  { symbol: 'SOL/USDT', exchange: 'Hyperliquid', pct: 12.4, pnl: 186 },
  { symbol: 'XAU/USD', exchange: 'MT5', pct: 4.8, pnl: 350 },
  { symbol: 'BTC/USDT', exchange: 'Binance', pct: 2.5, pnl: 85 },
  { symbol: 'ETH/USDT', exchange: 'Bybit', pct: -5.2, pnl: -84 },
  { symbol: 'EUR/USD', exchange: 'MT5', pct: -2.1, pnl: -105 },
  { symbol: 'AAPL', exchange: 'IBKR', pct: -0.8, pnl: -15 },
]

const best = movers.filter(m => m.pct > 0).sort((a, b) => b.pct - a.pct).slice(0, 3)
const worst = movers.filter(m => m.pct < 0).sort((a, b) => a.pct - b.pct).slice(0, 3)

function Sparkline({ up }: { up: boolean }) {
  // Simple procedural sparkline
  const pts: number[] = []
  let v = 50, s = up ? 42 : 99
  for (let i = 0; i < 20; i++) {
    s = (s * 16807 + 7) % 2147483647
    v += ((s % 100) / 100 - (up ? 0.4 : 0.6)) * 4
    v = Math.max(10, Math.min(90, v))
    pts.push(v)
  }
  const d = pts.map((p, i) => `${i === 0 ? 'M' : 'L'}${i * 3},${100 - p}`).join(' ')
  return (
    <svg viewBox="0 0 57 100" style={{ width: 40, height: 20 }} preserveAspectRatio="none">
      <path d={d} fill="none" stroke={up ? '#10b981' : '#ef4444'} strokeWidth="3" opacity="0.6" />
    </svg>
  )
}

function MoverRow({ symbol, exchange, pct, pnl }: typeof movers[0]) {
  const up = pct >= 0
  return (
    <div style={{
      display: 'flex', alignItems: 'center', justifyContent: 'space-between',
      padding: '8px 16px',
      borderBottom: '1px solid rgba(255,255,255,0.025)',
    }}>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 1, minWidth: 80 }}>
        <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-1)' }}>{symbol}</span>
        <span style={{ fontSize: 8, color: 'var(--text-3)' }}>{exchange}</span>
      </div>
      <Sparkline up={up} />
      <div style={{ textAlign: 'right' }}>
        <div className="mono" style={{ fontSize: 11, fontWeight: 700, color: up ? '#10b981' : '#ef4444' }}>
          {up ? '+' : ''}{pct}%
        </div>
        <div className="mono" style={{ fontSize: 9, color: up ? 'rgba(16,185,129,0.6)' : 'rgba(239,68,68,0.6)' }}>
          {up ? '+' : ''}${Math.abs(pnl)}
        </div>
      </div>
    </div>
  )
}

export default function TopMovers() {
  return (
    <div className="card fade-in" style={{ animationDelay: '250ms', display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      <div style={{ padding: '12px 20px', borderBottom: '1px solid var(--border)' }}>
        <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-1)' }}>Top Movers</span>
      </div>

      <div style={{ flex: 1, overflow: 'auto' }}>
        <div style={{ padding: '8px 16px 4px' }}>
          <span style={{ fontSize: 9, fontWeight: 600, color: '#10b981', textTransform: 'uppercase', letterSpacing: '0.06em' }}>Best Performers</span>
        </div>
        {best.map(m => <MoverRow key={m.symbol} {...m} />)}

        <div style={{ padding: '12px 16px 4px' }}>
          <span style={{ fontSize: 9, fontWeight: 600, color: '#ef4444', textTransform: 'uppercase', letterSpacing: '0.06em' }}>Worst Performers</span>
        </div>
        {worst.map(m => <MoverRow key={m.symbol} {...m} />)}
      </div>
    </div>
  )
}
