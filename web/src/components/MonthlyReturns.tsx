const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

// Procedural monthly returns
function genReturns(seed: number): number[][] {
  let s = seed
  const years: number[][] = []
  for (let y = 0; y < 2; y++) {
    const row: number[] = []
    for (let m = 0; m < 12; m++) {
      s = (s * 16807 + 13) % 2147483647
      const v = ((s % 2000) / 100 - 10) // -10% to +10%
      // Make current year partial (only up to March)
      if (y === 1 && m > 2) {
        row.push(NaN)
      } else {
        row.push(Math.round(v * 10) / 10)
      }
    }
    years.push(row)
  }
  return years
}

const data = genReturns(123)
const yearLabels = ['2025', '2026']

function getColor(v: number): string {
  if (isNaN(v)) return 'transparent'
  if (v >= 5) return 'rgba(16,185,129,0.5)'
  if (v >= 2) return 'rgba(16,185,129,0.3)'
  if (v >= 0) return 'rgba(16,185,129,0.12)'
  if (v >= -2) return 'rgba(239,68,68,0.12)'
  if (v >= -5) return 'rgba(239,68,68,0.3)'
  return 'rgba(239,68,68,0.5)'
}

function getTextColor(v: number): string {
  if (isNaN(v)) return 'transparent'
  if (v >= 2) return '#10b981'
  if (v >= 0) return 'rgba(16,185,129,0.7)'
  if (v >= -2) return 'rgba(239,68,68,0.7)'
  return '#ef4444'
}

export default function MonthlyReturns() {
  return (
    <div className="card fade-in" style={{ animationDelay: '200ms', display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      <div style={{ padding: '12px 20px', borderBottom: '1px solid var(--border)' }}>
        <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-1)' }}>Monthly Returns</span>
      </div>

      <div style={{ flex: 1, padding: 16, overflow: 'auto' }}>
        {/* Header */}
        <div style={{ display: 'grid', gridTemplateColumns: '40px repeat(12, 1fr)', gap: 3, marginBottom: 3 }}>
          <div />
          {months.map(m => (
            <div key={m} style={{ fontSize: 8, fontWeight: 600, color: 'var(--text-3)', textAlign: 'center', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
              {m}
            </div>
          ))}
        </div>

        {/* Rows */}
        {data.map((row, yi) => (
          <div key={yi} style={{ display: 'grid', gridTemplateColumns: '40px repeat(12, 1fr)', gap: 3, marginBottom: 3 }}>
            <div className="mono" style={{ fontSize: 9, fontWeight: 600, color: 'var(--text-3)', display: 'flex', alignItems: 'center' }}>
              {yearLabels[yi]}
            </div>
            {row.map((v, mi) => (
              <div key={mi} style={{
                background: getColor(v),
                borderRadius: 4,
                padding: '6px 0',
                textAlign: 'center',
                cursor: isNaN(v) ? 'default' : 'pointer',
                transition: 'transform 0.1s',
              }}
                title={isNaN(v) ? '' : `${months[mi]} ${yearLabels[yi]}: ${v >= 0 ? '+' : ''}${v}%`}
              >
                <span className="mono" style={{
                  fontSize: 9, fontWeight: 700,
                  color: getTextColor(v),
                }}>
                  {isNaN(v) ? '' : `${v >= 0 ? '+' : ''}${v}`}
                </span>
              </div>
            ))}
          </div>
        ))}

        {/* YTD row */}
        <div style={{ marginTop: 12, display: 'flex', gap: 12, justifyContent: 'center' }}>
          {data.map((row, yi) => {
            const valid = row.filter(v => !isNaN(v))
            const ytd = valid.reduce((s, v) => s + v, 0)
            const rounded = Math.round(ytd * 10) / 10
            return (
              <div key={yi} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <span style={{ fontSize: 9, color: 'var(--text-3)' }}>{yearLabels[yi]} YTD:</span>
                <span className="mono" style={{ fontSize: 11, fontWeight: 700, color: rounded >= 0 ? '#10b981' : '#ef4444' }}>
                  {rounded >= 0 ? '+' : ''}{rounded}%
                </span>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
