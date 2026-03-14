import { useState, useMemo, useRef, useCallback } from 'react'

const TF = ['1H', '4H', '1D', '1W'] as const

interface Candle { o: number; h: number; l: number; c: number; v: number; t: number }

function genCandles(count: number, base: number, vol: number, ms: number): Candle[] {
  const arr: Candle[] = []
  let p = base
  const now = Date.now()
  for (let i = 0; i < count; i++) {
    const d = (Math.random() - 0.46) * vol + Math.sin(i / 8) * vol * 0.08
    const o = p, c = o + d
    const w = Math.abs(d) * (0.3 + Math.random() * 0.7)
    arr.push({ o, h: Math.max(o, c) + w * Math.random(), l: Math.min(o, c) - w * Math.random(), c, v: 30 + Math.random() * 120, t: now - (count - i) * ms })
    p = c
  }
  return arr
}

const DATA: Record<string, Candle[]> = {
  '1H': genCandles(60, 69800, 150, 3600000),
  '4H': genCandles(48, 68500, 450, 14400000),
  '1D': genCandles(45, 65000, 1200, 86400000),
  '1W': genCandles(32, 55000, 3500, 604800000),
}

function fmtTime(ts: number, tf: string) {
  const d = new Date(ts)
  if (tf === '1H' || tf === '4H') return `${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}`
  return `${['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'][d.getMonth()]} ${d.getDate()}`
}

export default function PriceChart() {
  const [tf, setTf] = useState<typeof TF[number]>('1D')
  const [hov, setHov] = useState<{ idx: number; x: number; y: number } | null>(null)
  const ref = useRef<SVGSVGElement>(null)
  const candles = DATA[tf]

  const last = candles[candles.length - 1]
  const first = candles[0]
  const up = last.c >= first.o
  const price = last.c
  const pct = ((last.c - first.o) / first.o * 100).toFixed(2)

  const W = 800, H = 340
  const P = { t: 12, b: 28, l: 48, r: 60 }
  const cW = W - P.l - P.r, cH = H - P.t - P.b

  const { mn, mx, vm } = useMemo(() => {
    let mn = Infinity, mx = -Infinity, vm = 0
    for (const c of candles) { mn = Math.min(mn, c.l); mx = Math.max(mx, c.h); vm = Math.max(vm, c.v) }
    const pad = (mx - mn) * 0.05
    return { mn: mn - pad, mx: mx + pad, vm }
  }, [candles])

  const rng = mx - mn || 1
  const bw = cW / candles.length
  const bdy = Math.max(Math.min(bw * 0.55, 10), 2)
  const y = (v: number) => P.t + (1 - (v - mn) / rng) * cH

  const ema = useMemo(() => {
    const k = 2 / 10
    const r: number[] = []
    let prev = candles[0].c
    for (const c of candles) { prev = c.c * k + prev * (1 - k); r.push(prev) }
    return r
  }, [candles])

  const emaD = ema.map((v, i) => `${i ? 'L' : 'M'}${(P.l + i * bw + bw / 2).toFixed(1)},${y(v).toFixed(1)}`).join('')

  const grid = Array.from({ length: 5 }, (_, i) => {
    const v = mn + (rng * (i + 1)) / 6
    return { y: y(v), label: v >= 1000 ? `${(v / 1000).toFixed(1)}k` : v.toFixed(0) }
  })

  const tStep = Math.max(Math.floor(candles.length / 7), 1)

  const onMove = useCallback((e: React.MouseEvent<SVGSVGElement>) => {
    const svg = ref.current
    if (!svg) return
    const r = svg.getBoundingClientRect()
    const sx = W / r.width, sy = H / r.height
    const x = (e.clientX - r.left) * sx, yy = (e.clientY - r.top) * sy
    const idx = Math.floor((x - P.l) / bw)
    if (idx >= 0 && idx < candles.length && x >= P.l && x <= W - P.r) setHov({ idx, x, y: yy })
    else setHov(null)
  }, [candles.length, bw])

  const hc = hov ? candles[hov.idx] : null

  return (
    <div className="card fade-in" style={{ display: 'flex', flexDirection: 'column', height: '100%', animationDelay: '100ms' }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 20px', borderBottom: '1px solid var(--border)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ fontSize: 13, fontWeight: 700, color: 'var(--text-1)' }}>BTC/USDT</span>
            <span style={{ fontSize: 9, fontWeight: 600, padding: '2px 5px', borderRadius: 4, color: 'var(--text-3)', background: 'var(--bg-2)' }}>PERP</span>
          </div>
          <span className="mono" style={{ fontSize: 18, fontWeight: 700, color: 'var(--text-1)' }}>${price.toLocaleString(undefined, { maximumFractionDigits: 0 })}</span>
          <span className="mono" style={{ fontSize: 11, fontWeight: 600, color: up ? '#10b981' : '#ef4444' }}>{up ? '+' : ''}{pct}%</span>

          {hc && (
            <div className="mono" style={{ display: 'flex', gap: 10, fontSize: 10, color: 'var(--text-3)' }}>
              <span>O <span style={{ color: 'var(--text-2)' }}>{hc.o >= 1000 ? `${(hc.o/1000).toFixed(2)}k` : hc.o.toFixed(1)}</span></span>
              <span>H <span style={{ color: '#10b981' }}>{hc.h >= 1000 ? `${(hc.h/1000).toFixed(2)}k` : hc.h.toFixed(1)}</span></span>
              <span>L <span style={{ color: '#ef4444' }}>{hc.l >= 1000 ? `${(hc.l/1000).toFixed(2)}k` : hc.l.toFixed(1)}</span></span>
              <span>C <span style={{ color: hc.c >= hc.o ? '#10b981' : '#ef4444' }}>{hc.c >= 1000 ? `${(hc.c/1000).toFixed(2)}k` : hc.c.toFixed(1)}</span></span>
            </div>
          )}
        </div>

        <div style={{ display: 'flex', gap: 2, padding: 2, borderRadius: 6, background: 'var(--bg-2)' }}>
          {TF.map(t => (
            <button key={t} onClick={() => setTf(t)} style={{
              padding: '4px 10px', borderRadius: 4, border: 'none', cursor: 'pointer',
              fontSize: 11, fontWeight: 600,
              background: tf === t ? 'var(--bg-0)' : 'transparent',
              color: tf === t ? 'var(--text-1)' : 'var(--text-3)',
              transition: 'all 0.15s',
            }}>{t}</button>
          ))}
        </div>
      </div>

      {/* SVG */}
      <div style={{ flex: 1, padding: '4px 8px 8px', minHeight: 0, overflow: 'hidden' }}>
        <svg ref={ref} viewBox={`0 0 ${W} ${H}`} style={{ width: '100%', height: '100%' }} preserveAspectRatio="none" onMouseMove={onMove} onMouseLeave={() => setHov(null)}>
          {/* Grid */}
          {grid.map((g, i) => (
            <g key={i}>
              <line x1={P.l} y1={g.y} x2={W - P.r} y2={g.y} stroke="rgba(255,255,255,0.03)" />
              <text x={W - P.r + 6} y={g.y + 3} fill="rgba(255,255,255,0.2)" fontSize="9" fontFamily="JetBrains Mono">{g.label}</text>
            </g>
          ))}

          {/* Time */}
          {candles.map((c, i) => i % tStep === 0 ? (
            <text key={i} x={P.l + i * bw + bw / 2} y={H - 6} textAnchor="middle" fill="rgba(255,255,255,0.15)" fontSize="8" fontFamily="JetBrains Mono">{fmtTime(c.t, tf)}</text>
          ) : null)}

          {/* Volume */}
          {candles.map((c, i) => {
            const x = P.l + i * bw + bw / 2
            const vh = (c.v / vm) * cH * 0.12
            return <rect key={`v${i}`} x={x - bdy / 2} y={P.t + cH - vh} width={bdy} height={vh} fill={c.c >= c.o ? 'rgba(16,185,129,0.12)' : 'rgba(239,68,68,0.08)'} rx="0.5" />
          })}

          {/* EMA */}
          <path d={emaD} fill="none" stroke="rgba(99,102,241,0.4)" strokeWidth="1.2" />

          {/* Candles */}
          {candles.map((c, i) => {
            const x = P.l + i * bw + bw / 2
            const g = c.c >= c.o
            const top = y(Math.max(c.o, c.c)), bot = y(Math.min(c.o, c.c))
            const h = Math.max(bot - top, 1)
            return (
              <g key={i} opacity={hov?.idx === i ? 1 : 0.85}>
                <line x1={x} y1={y(c.h)} x2={x} y2={y(c.l)} stroke={g ? '#10b981' : '#ef4444'} strokeWidth="1" opacity="0.5" />
                <rect x={x - bdy / 2} y={top} width={bdy} height={h} fill={g ? '#10b981' : '#ef4444'} rx="0.5" opacity={g ? 0.85 : 0.7} />
              </g>
            )
          })}

          {/* Price line */}
          <line x1={P.l} y1={y(price)} x2={W - P.r} y2={y(price)} stroke={up ? '#10b981' : '#ef4444'} strokeDasharray="3,4" opacity="0.3" />
          <rect x={W - P.r + 1} y={y(price) - 9} width={P.r - 8} height={18} rx="3" fill={up ? '#10b981' : '#ef4444'} />
          <text x={W - P.r + (P.r - 8) / 2 + 1} y={y(price) + 3.5} textAnchor="middle" fill={up ? '#050810' : 'white'} fontSize="8.5" fontWeight="700" fontFamily="JetBrains Mono">
            {price >= 1000 ? `${(price / 1000).toFixed(2)}k` : price.toFixed(1)}
          </text>

          {/* Crosshair */}
          {hov && (
            <>
              <line x1={P.l + hov.idx * bw + bw / 2} y1={P.t} x2={P.l + hov.idx * bw + bw / 2} y2={P.t + cH} stroke="rgba(255,255,255,0.08)" strokeDasharray="2,3" />
              <line x1={P.l} y1={hov.y} x2={W - P.r} y2={hov.y} stroke="rgba(255,255,255,0.06)" strokeDasharray="2,3" />
            </>
          )}
        </svg>
      </div>
    </div>
  )
}
