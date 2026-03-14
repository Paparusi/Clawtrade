import { useState, useMemo } from 'react'

const ranges = ['1M', '3M', '6M', '1Y', 'ALL'] as const
const benchmarks = ['S&P 500', 'BTC', 'None'] as const

function generateEquity(days: number, seed: number, drift: number) {
  const pts: number[] = [10000]
  let s = seed
  for (let i = 1; i < days; i++) {
    s = (s * 16807 + 7) % 2147483647
    const r = ((s % 1000) / 1000 - 0.48) * 0.025 + drift
    pts.push(pts[i - 1] * (1 + r))
  }
  return pts
}

export default function PerformanceChart() {
  const [range, setRange] = useState<typeof ranges[number]>('6M')
  const [bench, setBench] = useState<typeof benchmarks[number]>('S&P 500')

  const days = { '1M': 30, '3M': 90, '6M': 180, '1Y': 365, 'ALL': 730 }[range]

  const { portfolio, benchmark, pMin, pMax } = useMemo(() => {
    const p = generateEquity(days, 42, 0.0004)
    const b = bench === 'None' ? null : generateEquity(days, bench === 'BTC' ? 99 : 77, bench === 'BTC' ? 0.0003 : 0.00025)
    const all = b ? [...p, ...b] : p
    return { portfolio: p, benchmark: b, pMin: Math.min(...all) * 0.98, pMax: Math.max(...all) * 1.02 }
  }, [days, bench])

  const W = 700, H = 280, pad = { t: 16, b: 28, l: 52, r: 16 }
  const cw = W - pad.l - pad.r, ch = H - pad.t - pad.b

  const toX = (i: number) => pad.l + (i / (portfolio.length - 1)) * cw
  const toY = (v: number) => pad.t + (1 - (v - pMin) / (pMax - pMin)) * ch

  const makePath = (data: number[]) => data.map((v, i) => `${i === 0 ? 'M' : 'L'}${toX(i).toFixed(1)},${toY(v).toFixed(1)}`).join(' ')
  const makeArea = (data: number[]) => makePath(data) + ` L${toX(data.length - 1)},${H - pad.b} L${pad.l},${H - pad.b} Z`

  const portfolioReturn = ((portfolio[portfolio.length - 1] / portfolio[0] - 1) * 100).toFixed(1)
  const benchReturn = benchmark ? ((benchmark[benchmark.length - 1] / benchmark[0] - 1) * 100).toFixed(1) : null

  // Grid lines
  const gridLines = 5
  const gridY = Array.from({ length: gridLines }, (_, i) => {
    const v = pMin + (pMax - pMin) * (i / (gridLines - 1))
    return { y: toY(v), label: `$${(v / 1000).toFixed(1)}k` }
  })

  return (
    <div className="card fade-in" style={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0, overflow: 'hidden' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 20px', borderBottom: '1px solid var(--border)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-1)' }}>Portfolio Performance</span>
          <div style={{ display: 'flex', gap: 2 }}>
            {benchmarks.map(b => (
              <button key={b} onClick={() => setBench(b)} style={{
                fontSize: 9, fontWeight: 600, padding: '2px 8px', borderRadius: 4, border: 'none', cursor: 'pointer',
                background: bench === b ? 'rgba(99,102,241,0.15)' : 'transparent',
                color: bench === b ? '#818cf8' : 'var(--text-3)',
              }}>{b}</button>
            ))}
          </div>
        </div>
        <div style={{ display: 'flex', gap: 2 }}>
          {ranges.map(r => (
            <button key={r} onClick={() => setRange(r)} style={{
              fontSize: 9, fontWeight: 600, padding: '3px 8px', borderRadius: 4, border: 'none', cursor: 'pointer',
              background: range === r ? 'rgba(99,102,241,0.15)' : 'transparent',
              color: range === r ? '#818cf8' : 'var(--text-3)',
            }}>{r}</button>
          ))}
        </div>
      </div>

      <div style={{ flex: 1, padding: '4px 8px 8px', minHeight: 0, overflow: 'hidden' }}>
        <svg viewBox={`0 0 ${W} ${H}`} style={{ width: '100%', height: '100%' }} preserveAspectRatio="none">
          {/* Grid */}
          {gridY.map((g, i) => (
            <g key={i}>
              <line x1={pad.l} y1={g.y} x2={W - pad.r} y2={g.y} stroke="rgba(255,255,255,0.04)" strokeWidth="1" />
              <text x={pad.l - 6} y={g.y + 3} fill="#52525b" fontSize="9" textAnchor="end" fontFamily="JetBrains Mono">{g.label}</text>
            </g>
          ))}

          {/* Benchmark */}
          {benchmark && <>
            <path d={makeArea(benchmark)} fill="rgba(161,161,170,0.04)" />
            <path d={makePath(benchmark)} fill="none" stroke="rgba(161,161,170,0.35)" strokeWidth="1.5" />
          </>}

          {/* Portfolio */}
          <path d={makeArea(portfolio)} fill="rgba(99,102,241,0.08)" />
          <path d={makePath(portfolio)} fill="none" stroke="#6366f1" strokeWidth="2" />

          {/* Current value dot */}
          <circle cx={toX(portfolio.length - 1)} cy={toY(portfolio[portfolio.length - 1])} r="3" fill="#6366f1" />
        </svg>
      </div>

      {/* Legend */}
      <div style={{ display: 'flex', gap: 20, padding: '0 20px 12px', alignItems: 'center' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <div style={{ width: 12, height: 2, background: '#6366f1', borderRadius: 1 }} />
          <span className="mono" style={{ fontSize: 10, color: 'var(--text-2)' }}>Portfolio</span>
          <span className="mono" style={{ fontSize: 10, fontWeight: 700, color: Number(portfolioReturn) >= 0 ? '#10b981' : '#ef4444' }}>
            {Number(portfolioReturn) >= 0 ? '+' : ''}{portfolioReturn}%
          </span>
        </div>
        {benchmark && benchReturn && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <div style={{ width: 12, height: 2, background: 'rgba(161,161,170,0.5)', borderRadius: 1 }} />
            <span className="mono" style={{ fontSize: 10, color: 'var(--text-3)' }}>{bench}</span>
            <span className="mono" style={{ fontSize: 10, fontWeight: 700, color: Number(benchReturn) >= 0 ? '#10b981' : '#ef4444' }}>
              {Number(benchReturn) >= 0 ? '+' : ''}{benchReturn}%
            </span>
          </div>
        )}
      </div>
    </div>
  )
}
