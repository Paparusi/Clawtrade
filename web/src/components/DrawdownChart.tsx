import { useMemo } from 'react'

function generateDrawdown(days: number, seed: number): number[] {
  const dd: number[] = []
  let peak = 10000, equity = 10000, s = seed
  for (let i = 0; i < days; i++) {
    s = (s * 16807 + 7) % 2147483647
    const r = ((s % 1000) / 1000 - 0.48) * 0.025 + 0.0004
    equity *= (1 + r)
    if (equity > peak) peak = equity
    dd.push(((equity - peak) / peak) * 100)
  }
  return dd
}

export default function DrawdownChart() {
  const days = 180
  const data = useMemo(() => generateDrawdown(days, 42), [])

  const W = 900, H = 180, pad = { t: 12, b: 24, l: 48, r: 16 }
  const cw = W - pad.l - pad.r, ch = H - pad.t - pad.b

  const minDD = Math.min(...data)
  const maxDD = 0

  const toX = (i: number) => pad.l + (i / (data.length - 1)) * cw
  const toY = (v: number) => pad.t + (1 - (v - minDD) / (maxDD - minDD)) * ch

  const areaPath = data.map((v, i) => `${i === 0 ? 'M' : 'L'}${toX(i).toFixed(1)},${toY(v).toFixed(1)}`).join(' ')
    + ` L${toX(data.length - 1)},${toY(0)} L${pad.l},${toY(0)} Z`
  const linePath = data.map((v, i) => `${i === 0 ? 'M' : 'L'}${toX(i).toFixed(1)},${toY(v).toFixed(1)}`).join(' ')

  // Max drawdown annotation
  const maxIdx = data.indexOf(minDD)
  const maxDDLabel = `${minDD.toFixed(1)}%`

  // Grid
  const gridSteps = [0, -3, -6, -9, -12, -15].filter(v => v >= minDD * 1.1)

  return (
    <div className="card fade-in" style={{ animationDelay: '300ms', display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0, overflow: 'hidden' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 20px', borderBottom: '1px solid var(--border)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-1)' }}>Drawdown</span>
          <span className="mono" style={{
            fontSize: 10, fontWeight: 700, padding: '1px 6px', borderRadius: 4,
            color: '#ef4444', background: 'rgba(239,68,68,0.08)',
          }}>Max: {maxDDLabel}</span>
        </div>
        <span style={{ fontSize: 10, color: 'var(--text-3)' }}>Last 6 months</span>
      </div>

      <div style={{ flex: 1, padding: '4px 8px 8px', minHeight: 0, overflow: 'hidden' }}>
        <svg viewBox={`0 0 ${W} ${H}`} style={{ width: '100%', height: '100%' }} preserveAspectRatio="none">
          {/* Zero line */}
          <line x1={pad.l} y1={toY(0)} x2={W - pad.r} y2={toY(0)} stroke="rgba(255,255,255,0.08)" strokeWidth="1" />

          {/* Grid */}
          {gridSteps.map((v, i) => (
            <g key={i}>
              <line x1={pad.l} y1={toY(v)} x2={W - pad.r} y2={toY(v)} stroke="rgba(255,255,255,0.03)" strokeWidth="1" />
              <text x={pad.l - 6} y={toY(v) + 3} fill="#52525b" fontSize="9" textAnchor="end" fontFamily="JetBrains Mono">{v}%</text>
            </g>
          ))}

          {/* Area */}
          <path d={areaPath} fill="rgba(239,68,68,0.1)" />
          <path d={linePath} fill="none" stroke="#ef4444" strokeWidth="1.5" opacity="0.7" />

          {/* Max drawdown point */}
          <circle cx={toX(maxIdx)} cy={toY(minDD)} r="3" fill="#ef4444" />
          <line x1={toX(maxIdx)} y1={toY(minDD) - 8} x2={toX(maxIdx)} y2={toY(0)} stroke="#ef4444" strokeWidth="0.5" strokeDasharray="3,3" opacity="0.4" />
        </svg>
      </div>
    </div>
  )
}
