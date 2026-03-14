const segments = [
  { label: 'Crypto', pct: 48.5, value: 4970, color: '#6366f1' },
  { label: 'Forex', pct: 24.2, value: 2480, color: '#06b6d4' },
  { label: 'Stocks', pct: 18.1, value: 1855, color: '#10b981' },
  { label: 'DeFi', pct: 9.2, value: 941, color: '#f59e0b' },
]

export default function AssetAllocation() {
  const total = segments.reduce((s, seg) => s + seg.value, 0)

  // SVG donut chart
  const cx = 80, cy = 80, r = 60, stroke = 16
  let cumAngle = -90

  const arcs = segments.map(seg => {
    const angle = (seg.pct / 100) * 360
    const startAngle = cumAngle
    cumAngle += angle

    const startRad = (startAngle * Math.PI) / 180
    const endRad = ((startAngle + angle) * Math.PI) / 180

    const x1 = cx + r * Math.cos(startRad)
    const y1 = cy + r * Math.sin(startRad)
    const x2 = cx + r * Math.cos(endRad)
    const y2 = cy + r * Math.sin(endRad)

    const largeArc = angle > 180 ? 1 : 0

    return {
      ...seg,
      d: `M ${x1} ${y1} A ${r} ${r} 0 ${largeArc} 1 ${x2} ${y2}`,
    }
  })

  return (
    <div className="card fade-in" style={{ animationDelay: '150ms', display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      <div style={{ padding: '12px 20px', borderBottom: '1px solid var(--border)' }}>
        <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-1)' }}>Asset Allocation</span>
      </div>

      <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 16, gap: 24 }}>
        {/* Donut */}
        <div style={{ position: 'relative', width: 160, height: 160, flexShrink: 0 }}>
          <svg viewBox="0 0 160 160" style={{ width: '100%', height: '100%' }}>
            {arcs.map((arc, i) => (
              <path key={i} d={arc.d} fill="none" stroke={arc.color} strokeWidth={stroke} strokeLinecap="butt" />
            ))}
          </svg>
          <div style={{
            position: 'absolute', inset: 0, display: 'flex', flexDirection: 'column',
            alignItems: 'center', justifyContent: 'center',
          }}>
            <span className="mono" style={{ fontSize: 16, fontWeight: 800, color: 'var(--text-1)' }}>
              ${(total / 1000).toFixed(1)}k
            </span>
            <span style={{ fontSize: 9, color: 'var(--text-3)' }}>Total</span>
          </div>
        </div>

        {/* Legend */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {segments.map(seg => (
            <div key={seg.label} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <div style={{ width: 8, height: 8, borderRadius: 2, background: seg.color, flexShrink: 0 }} />
              <div style={{ minWidth: 50 }}>
                <span style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-2)' }}>{seg.label}</span>
              </div>
              <span className="mono" style={{ fontSize: 11, fontWeight: 700, color: 'var(--text-1)' }}>{seg.pct}%</span>
              <span className="mono" style={{ fontSize: 10, color: 'var(--text-3)' }}>${seg.value.toLocaleString()}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
