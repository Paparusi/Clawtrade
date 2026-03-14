import { useState, useEffect, useMemo } from 'react'

const signals = [
  { time: '14:32', action: 'BUY', sym: 'BTC/USDT', exchange: 'Binance', reason: 'EMA crossover bullish + RSI oversold bounce at 32.4', confidence: 87, pnl: '+$85' },
  { time: '14:15', action: 'SELL', sym: 'EUR/USD', exchange: 'MT5', reason: 'Bearish divergence on H4 + USD strength from DXY breakout', confidence: 74, pnl: '+$225' },
  { time: '13:48', action: 'CLOSE', sym: 'SOL/USDT', exchange: 'Hyperliquid', reason: 'TP hit at 3.47% — trailing stop activated', confidence: 92, pnl: '+$93' },
  { time: '13:15', action: 'BUY', sym: 'AAPL', exchange: 'IBKR', reason: 'Pre-earnings momentum + sector rotation into tech', confidence: 68, pnl: null },
  { time: '12:30', action: 'HOLD', sym: 'XAU/USD', exchange: 'MT5', reason: 'Consolidating near $2,350 resistance — waiting for confirmation', confidence: 55, pnl: null },
]

const actionStyle: Record<string, { color: string; bg: string }> = {
  BUY: { color: '#10b981', bg: 'rgba(16,185,129,0.1)' },
  SELL: { color: '#ef4444', bg: 'rgba(239,68,68,0.1)' },
  CLOSE: { color: '#f59e0b', bg: 'rgba(245,158,11,0.1)' },
  HOLD: { color: '#6366f1', bg: 'rgba(99,102,241,0.1)' },
}

function MiniPerformance() {
  const data = useMemo(() => {
    const pts: number[] = []
    let v = 10000
    for (let i = 0; i < 30; i++) {
      v += (Math.random() - 0.38) * 80
      pts.push(v)
    }
    return pts
  }, [])

  const min = Math.min(...data), max = Math.max(...data)
  const rng = max - min || 1
  const w = 200, h = 40
  const pts = data.map((v, i) => `${(i / (data.length - 1)) * w},${h - ((v - min) / rng) * h}`).join(' ')
  const area = `0,${h} ${pts} ${w},${h}`
  const up = data[data.length - 1] >= data[0]
  const c = up ? '#10b981' : '#ef4444'

  return (
    <svg width={w} height={h} viewBox={`0 0 ${w} ${h}`} style={{ display: 'block' }}>
      <defs>
        <linearGradient id="perfGrad" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={c} stopOpacity="0.15" />
          <stop offset="100%" stopColor={c} stopOpacity="0" />
        </linearGradient>
      </defs>
      <polygon points={area} fill="url(#perfGrad)" />
      <polyline points={pts} fill="none" stroke={c} strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

export default function AgentStatus() {
  const [dots, setDots] = useState(1)

  useEffect(() => {
    const i = setInterval(() => setDots(d => d >= 3 ? 1 : d + 1), 600)
    return () => clearInterval(i)
  }, [])

  return (
    <div className="card fade-in" style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Agent Header */}
      <div style={{ padding: '20px', borderBottom: '1px solid var(--border)' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            {/* Agent avatar with ring */}
            <div style={{ position: 'relative' }}>
              <div style={{
                width: 40, height: 40, borderRadius: 12,
                background: 'linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%)',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                boxShadow: '0 0 20px rgba(99,102,241,0.2)',
              }}>
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none">
                  <path d="M12 2a8 8 0 0 1 8 8v1a8 8 0 0 1-16 0v-1a8 8 0 0 1 8-8z" stroke="white" strokeWidth="1.5" fill="none"/>
                  <circle cx="9" cy="9.5" r="1.2" fill="white"/>
                  <circle cx="15" cy="9.5" r="1.2" fill="white"/>
                  <path d="M9 14c1.5 1.2 4.5 1.2 6 0" stroke="white" strokeWidth="1.2" strokeLinecap="round"/>
                </svg>
              </div>
              <div style={{
                position: 'absolute', bottom: -2, right: -2,
                width: 12, height: 12, borderRadius: '50%',
                background: '#10b981', border: '2px solid var(--bg-1)',
              }} />
            </div>
            <div>
              <div style={{ fontSize: 14, fontWeight: 700, color: 'var(--text-1)' }}>Claw Agent</div>
              <div style={{ fontSize: 11, color: 'var(--text-3)', display: 'flex', alignItems: 'center', gap: 4 }}>
                <span>Scanning 6 exchanges</span>
                <span className="mono" style={{ color: 'var(--accent-2)' }}>{'.'.repeat(dots)}</span>
              </div>
            </div>
          </div>
          <div style={{
            padding: '4px 10px', borderRadius: 6,
            fontSize: 10, fontWeight: 600,
            color: '#10b981', background: 'rgba(16,185,129,0.08)',
            border: '1px solid rgba(16,185,129,0.12)',
          }}>Running</div>
        </div>

        {/* Performance chart + stats */}
        <div style={{
          background: 'var(--bg-2)', borderRadius: 10, padding: '14px 16px',
          border: '1px solid var(--border)',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10 }}>
            <div>
              <div style={{ fontSize: 10, color: 'var(--text-3)', marginBottom: 2 }}>7-Day Performance</div>
              <div className="mono" style={{ fontSize: 18, fontWeight: 700, color: '#10b981' }}>+$1,247</div>
            </div>
            <div style={{ display: 'flex', gap: 16, fontSize: 10 }}>
              <div style={{ textAlign: 'center' }}>
                <div className="mono" style={{ fontSize: 14, fontWeight: 700, color: 'var(--text-1)' }}>73%</div>
                <div style={{ color: 'var(--text-3)', marginTop: 1 }}>Accuracy</div>
              </div>
              <div style={{ textAlign: 'center' }}>
                <div className="mono" style={{ fontSize: 14, fontWeight: 700, color: 'var(--text-1)' }}>23</div>
                <div style={{ color: 'var(--text-3)', marginTop: 1 }}>Trades</div>
              </div>
              <div style={{ textAlign: 'center' }}>
                <div className="mono" style={{ fontSize: 14, fontWeight: 700, color: '#f59e0b' }}>1.85</div>
                <div style={{ color: 'var(--text-3)', marginTop: 1 }}>R:R</div>
              </div>
            </div>
          </div>
          <MiniPerformance />
        </div>
      </div>

      {/* Signals */}
      <div style={{ padding: '14px 20px 8px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-1)' }}>Recent Signals</span>
        <span style={{ fontSize: 10, color: 'var(--text-3)' }}>Today</span>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '0 12px 12px' }}>
        {signals.map((s, i) => {
          const st = actionStyle[s.action]
          return (
            <div
              key={i}
              style={{
                padding: '14px', marginBottom: 6, borderRadius: 10,
                background: 'var(--bg-2)', border: '1px solid var(--border)',
                cursor: 'pointer', transition: 'all 0.15s',
              }}
              onMouseEnter={e => { e.currentTarget.style.borderColor = 'var(--border-hover)'; e.currentTarget.style.transform = 'translateY(-1px)' }}
              onMouseLeave={e => { e.currentTarget.style.borderColor = 'var(--border)'; e.currentTarget.style.transform = 'translateY(0)' }}
            >
              {/* Top row */}
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span style={{
                    fontSize: 9, fontWeight: 700, padding: '2px 6px', borderRadius: 4,
                    color: st.color, background: st.bg, letterSpacing: '0.04em',
                  }}>{s.action}</span>
                  <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--text-1)' }}>{s.sym}</span>
                  <span style={{ fontSize: 9, color: 'var(--text-3)' }}>{s.exchange}</span>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  {s.pnl && <span className="mono" style={{ fontSize: 11, fontWeight: 600, color: '#10b981' }}>{s.pnl}</span>}
                  <span className="mono" style={{ fontSize: 10, color: 'var(--text-3)' }}>{s.time}</span>
                </div>
              </div>

              {/* Reason */}
              <div style={{ fontSize: 11, color: 'var(--text-2)', lineHeight: 1.5, marginBottom: 10 }}>
                {s.reason}
              </div>

              {/* Confidence */}
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <div style={{ flex: 1, height: 4, borderRadius: 2, background: 'var(--bg-3)', overflow: 'hidden' }}>
                  <div style={{
                    width: `${s.confidence}%`, height: '100%', borderRadius: 2,
                    background: s.confidence >= 80 ? '#10b981' : s.confidence >= 60 ? '#f59e0b' : '#ef4444',
                    boxShadow: `0 0 8px ${s.confidence >= 80 ? 'rgba(16,185,129,0.3)' : s.confidence >= 60 ? 'rgba(245,158,11,0.3)' : 'rgba(239,68,68,0.3)'}`,
                  }} />
                </div>
                <span className="mono" style={{
                  fontSize: 11, fontWeight: 700, minWidth: 32, textAlign: 'right',
                  color: s.confidence >= 80 ? '#10b981' : s.confidence >= 60 ? '#f59e0b' : '#ef4444',
                }}>{s.confidence}%</span>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
