import { useState } from 'react'

type AC = 'all' | 'crypto' | 'forex' | 'stocks' | 'index'

const markets = [
  { sym: 'BTC/USDT', price: 70245, change: 2.34, exchange: 'Binance', class: 'crypto' as AC },
  { sym: 'ETH/USDT', price: 3382, change: -1.12, exchange: 'Binance', class: 'crypto' as AC },
  { sym: 'SOL/USDT', price: 172.3, change: 5.67, exchange: 'Bybit', class: 'crypto' as AC },
  { sym: 'BNB/USDT', price: 612.4, change: 0.89, exchange: 'Binance', class: 'crypto' as AC },
  { sym: 'EUR/USD', price: 1.0847, change: -0.12, exchange: 'MT5', class: 'forex' as AC },
  { sym: 'GBP/JPY', price: 196.42, change: 0.34, exchange: 'MT5', class: 'forex' as AC },
  { sym: 'XAU/USD', price: 2345, change: 0.89, exchange: 'MT5', class: 'forex' as AC },
  { sym: 'AAPL', price: 198.52, change: 1.05, exchange: 'IBKR', class: 'stocks' as AC },
  { sym: 'NVDA', price: 875.3, change: 3.42, exchange: 'IBKR', class: 'stocks' as AC },
  { sym: 'TSLA', price: 245.6, change: -2.15, exchange: 'IBKR', class: 'stocks' as AC },
  { sym: 'NQ100', price: 18420, change: 0.67, exchange: 'CME', class: 'index' as AC },
  { sym: 'SPX', price: 5234, change: 0.42, exchange: 'CME', class: 'index' as AC },
  { sym: 'VIX', price: 14.8, change: -5.12, exchange: 'CME', class: 'index' as AC },
]

const tabs: { id: AC; label: string }[] = [
  { id: 'all', label: 'All' },
  { id: 'crypto', label: 'Crypto' },
  { id: 'forex', label: 'Forex' },
  { id: 'stocks', label: 'Stocks' },
  { id: 'index', label: 'Index' },
]

function fmt(p: number, s: string) {
  if (p >= 10000) return `$${(p / 1000).toFixed(1)}k`
  if (p >= 100) return `$${p.toLocaleString()}`
  if (s.includes('/') && p < 10) return p.toFixed(4)
  return `$${p.toFixed(1)}`
}

export default function MarketOverview() {
  const [tab, setTab] = useState<AC>('all')
  const filtered = tab === 'all' ? markets : markets.filter(m => m.class === tab)

  return (
    <div className="card fade-in" style={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0, animationDelay: '200ms' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 20px', borderBottom: '1px solid var(--border)' }}>
        <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-1)' }}>Markets</span>
        <div style={{ display: 'flex', gap: 2 }}>
          {tabs.map(t => (
            <button key={t.id} onClick={() => setTab(t.id)} style={{
              padding: '3px 8px', borderRadius: 4, border: 'none', cursor: 'pointer',
              fontSize: 10, fontWeight: 600,
              background: tab === t.id ? 'rgba(99,102,241,0.1)' : 'transparent',
              color: tab === t.id ? '#818cf8' : 'var(--text-3)',
              transition: 'all 0.15s',
            }}>{t.label}</button>
          ))}
        </div>
      </div>

      <div style={{ flex: 1, overflow: 'auto' }}>
        {filtered.map(m => {
          const up = m.change >= 0
          return (
            <div
              key={m.sym}
              style={{
                display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                padding: '10px 20px', cursor: 'pointer',
                borderBottom: '1px solid rgba(255,255,255,0.025)',
                transition: 'background 0.15s',
              }}
              onMouseEnter={e => e.currentTarget.style.background = 'rgba(255,255,255,0.02)'}
              onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-1)' }}>{m.sym}</span>
                <span style={{ fontSize: 9, color: 'var(--text-3)' }}>{m.exchange}</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                <span className="mono" style={{ fontSize: 12, color: 'var(--text-1)', fontWeight: 500 }}>{fmt(m.price, m.sym)}</span>
                <span className="mono" style={{
                  fontSize: 11, fontWeight: 600, minWidth: 52, textAlign: 'right',
                  color: up ? '#10b981' : '#ef4444',
                }}>{up ? '+' : ''}{m.change}%</span>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
