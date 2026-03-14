import { useState, useEffect } from 'react'
import { fetchPositions, type PositionData } from '../api/client'

interface Position {
  symbol: string
  exchange: string
  type: string
  side: string
  size: string
  entry: number
  mark: number
  pnl: number
  pnlPct: number
  lev: string
}

const FALLBACK: Position[] = [
  { symbol: 'BTC/USDT', exchange: 'Binance', type: 'Crypto', side: 'Long', size: '0.05 BTC', entry: 68500, mark: 70200, pnl: 85, pnlPct: 2.48, lev: '10x' },
  { symbol: 'EUR/USD', exchange: 'MT5', type: 'Forex', side: 'Short', size: '0.5 lot', entry: 1.0892, mark: 1.0847, pnl: 225, pnlPct: 0.41, lev: '100x' },
  { symbol: 'ETH/USDT', exchange: 'Bybit', type: 'Crypto', side: 'Long', size: '1.2 ETH', entry: 3450, mark: 3380, pnl: -84, pnlPct: -2.03, lev: '5x' },
  { symbol: 'AAPL', exchange: 'IBKR', type: 'Stock', side: 'Long', size: '25 shares', entry: 195.4, mark: 198.52, pnl: 78, pnlPct: 1.60, lev: '1x' },
  { symbol: 'SOL/USDT', exchange: 'Hyperliquid', type: 'Crypto', side: 'Short', size: '15 SOL', entry: 178.5, mark: 172.3, pnl: 93, pnlPct: 3.47, lev: '3x' },
  { symbol: 'XAU/USD', exchange: 'MT5', type: 'Forex', side: 'Long', size: '0.1 lot', entry: 2310, mark: 2345, pnl: 350, pnlPct: 1.52, lev: '50x' },
]

function apiToPosition(p: PositionData): Position {
  const pnlPct = p.entry_price > 0 ? ((p.current_price - p.entry_price) / p.entry_price * 100) : 0
  return {
    symbol: p.symbol,
    exchange: p.exchange,
    type: 'Crypto',
    side: p.side === 'long' ? 'Long' : 'Short',
    size: `${p.size}`,
    entry: p.entry_price,
    mark: p.current_price,
    pnl: p.pnl,
    pnlPct: +pnlPct.toFixed(2),
    lev: '1x',
  }
}

export default function PositionsTable() {
  const [positions, setPositions] = useState<Position[]>(FALLBACK)

  useEffect(() => {
    let cancelled = false
    fetchPositions()
      .then(data => {
        if (cancelled || !data.length) return
        setPositions(data.map(apiToPosition))
      })
      .catch(() => {})
    return () => { cancelled = true }
  }, [])

  const total = positions.reduce((s, p) => s + p.pnl, 0)

  return (
    <div className="card fade-in" style={{ animationDelay: '200ms', display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0, overflow: 'hidden' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 20px', borderBottom: '1px solid var(--border)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-1)' }}>Positions</span>
          <span style={{ fontSize: 10, fontWeight: 600, color: 'var(--text-3)', background: 'var(--bg-2)', padding: '1px 6px', borderRadius: 4 }}>{positions.length}</span>
        </div>
        <span className="mono" style={{ fontSize: 12, fontWeight: 700, color: total >= 0 ? '#10b981' : '#ef4444' }}>
          {total >= 0 ? '+' : ''}${total}
        </span>
      </div>

      <div style={{ flex: 1, overflow: 'auto', minHeight: 0 }}>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr style={{ fontSize: 10, color: 'var(--text-3)', fontWeight: 500, textTransform: 'uppercase', letterSpacing: '0.04em' }}>
            <th style={{ textAlign: 'left', padding: '8px 20px', fontWeight: 500 }}>Symbol</th>
            <th style={{ textAlign: 'left', padding: '8px 12px', fontWeight: 500 }}>Side</th>
            <th style={{ textAlign: 'right', padding: '8px 12px', fontWeight: 500 }}>Size</th>
            <th style={{ textAlign: 'center', padding: '8px 12px', fontWeight: 500 }}>Leverage</th>
            <th style={{ textAlign: 'right', padding: '8px 12px', fontWeight: 500 }}>Entry</th>
            <th style={{ textAlign: 'right', padding: '8px 12px', fontWeight: 500 }}>Mark</th>
            <th style={{ textAlign: 'right', padding: '8px 20px', fontWeight: 500 }}>PnL</th>
          </tr>
        </thead>
        <tbody>
          {positions.map((p) => {
            const isUp = p.pnl >= 0
            return (
              <tr
                key={`${p.symbol}-${p.exchange}`}
                style={{ borderTop: '1px solid rgba(255,255,255,0.025)', cursor: 'pointer', transition: 'background 0.15s' }}
                onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255,255,255,0.015)'}
                onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
              >
                <td style={{ padding: '10px 20px' }}>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                    <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-1)' }}>{p.symbol}</span>
                    <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                      <span style={{ fontSize: 9, color: 'var(--text-3)' }}>{p.exchange}</span>
                      <span style={{ fontSize: 8, fontWeight: 600, padding: '0 4px', borderRadius: 3, color: 'var(--text-3)', background: 'var(--bg-2)' }}>{p.type}</span>
                    </div>
                  </div>
                </td>
                <td style={{ padding: '10px 12px' }}>
                  <span style={{
                    fontSize: 10, fontWeight: 600, padding: '2px 6px', borderRadius: 4,
                    color: p.side === 'Long' ? '#10b981' : '#ef4444',
                    background: p.side === 'Long' ? 'rgba(16,185,129,0.08)' : 'rgba(239,68,68,0.08)',
                  }}>{p.side}</span>
                </td>
                <td className="mono" style={{ padding: '10px 12px', textAlign: 'right', fontSize: 11, color: 'var(--text-2)' }}>{p.size}</td>
                <td className="mono" style={{ padding: '10px 12px', textAlign: 'center', fontSize: 10, color: 'var(--text-3)' }}>{p.lev}</td>
                <td className="mono" style={{ padding: '10px 12px', textAlign: 'right', fontSize: 11, color: 'var(--text-3)' }}>
                  {p.entry >= 1000 ? `$${p.entry.toLocaleString()}` : p.entry < 10 ? p.entry.toFixed(4) : `$${p.entry}`}
                </td>
                <td className="mono" style={{ padding: '10px 12px', textAlign: 'right', fontSize: 11, color: 'var(--text-1)', fontWeight: 500 }}>
                  {p.mark >= 1000 ? `$${p.mark.toLocaleString()}` : p.mark < 10 ? p.mark.toFixed(4) : `$${p.mark}`}
                </td>
                <td style={{ padding: '10px 20px', textAlign: 'right' }}>
                  <div className="mono" style={{ fontSize: 12, fontWeight: 700, color: isUp ? '#10b981' : '#ef4444' }}>
                    {isUp ? '+' : ''}${Math.abs(p.pnl)}
                  </div>
                  <div className="mono" style={{ fontSize: 9, color: isUp ? 'rgba(16,185,129,0.6)' : 'rgba(239,68,68,0.6)' }}>
                    {isUp ? '+' : ''}{p.pnlPct}%
                  </div>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
      </div>
    </div>
  )
}
