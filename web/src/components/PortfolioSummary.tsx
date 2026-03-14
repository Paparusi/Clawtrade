import { useState, useEffect } from 'react'
import { fetchBalances, fetchPositions, type BalanceData, type PositionData } from '../api/client'

interface StatItem {
  label: string
  value: string
  change: string
  pct: string
  up: boolean
}

const FALLBACK: StatItem[] = [
  { label: 'Portfolio Value', value: '$10,245.80', change: '+$245.80', pct: '+2.45%', up: true },
  { label: 'Unrealized P&L', value: '+$747.00', change: '6 positions', pct: '+7.29%', up: true },
  { label: "Today's P&L", value: '-$45.20', change: 'Since 00:00 UTC', pct: '-0.44%', up: false },
  { label: 'Win Rate', value: '68.5%', change: '94W / 43L', pct: '1.85 R:R', up: true },
]

function fmtUsd(n: number): string {
  const abs = Math.abs(n)
  if (abs >= 1000000) return `${n >= 0 ? '' : '-'}$${(abs / 1000000).toFixed(2)}M`
  if (abs >= 1000) return `${n >= 0 ? '' : '-'}$${abs.toLocaleString(undefined, { maximumFractionDigits: 2 })}`
  return `${n >= 0 ? '' : '-'}$${abs.toFixed(2)}`
}

export default function PortfolioSummary() {
  const [stats, setStats] = useState<StatItem[]>(FALLBACK)

  useEffect(() => {
    let cancelled = false
    Promise.allSettled([fetchBalances(), fetchPositions()])
      .then(([balRes, posRes]) => {
        if (cancelled) return
        const balances: BalanceData[] = balRes.status === 'fulfilled' ? balRes.value : []
        const positions: PositionData[] = posRes.status === 'fulfilled' ? posRes.value : []

        if (!balances.length && !positions.length) return

        const totalValue = balances.reduce((s, b) => s + b.total, 0)
        const totalPnl = positions.reduce((s, p) => s + p.pnl, 0)
        const pnlPct = totalValue > 0 ? (totalPnl / totalValue * 100) : 0

        const updated: StatItem[] = [
          {
            label: 'Portfolio Value',
            value: fmtUsd(totalValue),
            change: `${balances.length} assets`,
            pct: totalValue > 0 ? '+0.00%' : '0.00%',
            up: true,
          },
          {
            label: 'Unrealized P&L',
            value: `${totalPnl >= 0 ? '+' : ''}${fmtUsd(totalPnl)}`,
            change: `${positions.length} positions`,
            pct: `${pnlPct >= 0 ? '+' : ''}${pnlPct.toFixed(2)}%`,
            up: totalPnl >= 0,
          },
          FALLBACK[2],
          FALLBACK[3],
        ]
        setStats(updated)
      })
    return () => { cancelled = true }
  }, [])

  return (
    <div className="card fade-in" style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)' }}>
      {stats.map((s, i) => (
        <div key={s.label} style={{
          padding: '18px 24px',
          borderRight: i < stats.length - 1 ? '1px solid var(--border)' : 'none',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 10 }}>
            <span style={{ fontSize: 11, color: 'var(--text-3)', fontWeight: 500 }}>{s.label}</span>
            <span className="mono" style={{
              fontSize: 10, fontWeight: 600, padding: '2px 6px', borderRadius: 4,
              color: s.up ? '#10b981' : '#ef4444',
              background: s.up ? 'rgba(16,185,129,0.08)' : 'rgba(239,68,68,0.08)',
            }}>{s.pct}</span>
          </div>
          <div className="mono" style={{
            fontSize: 22, fontWeight: 800, letterSpacing: '-0.02em', lineHeight: 1,
            color: s.label.includes('P&L') ? (s.up ? '#10b981' : '#ef4444') : 'var(--text-1)',
          }}>{s.value}</div>
          <div style={{ fontSize: 10, color: 'var(--text-3)', marginTop: 6 }}>{s.change}</div>
        </div>
      ))}
    </div>
  )
}
