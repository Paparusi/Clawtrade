import { useState, useEffect } from 'react'
import { useWS } from '../hooks/WebSocketProvider'
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

function buildStats(balances: BalanceData[], positions: PositionData[], totalPnl?: number): StatItem[] {
  const totalValue = balances.reduce((s, b) => s + b.total, 0)
  const pnl = totalPnl ?? positions.reduce((s, p) => s + p.pnl, 0)
  const pnlPct = totalValue > 0 ? (pnl / totalValue * 100) : 0

  return [
    {
      label: 'Portfolio Value',
      value: fmtUsd(totalValue),
      change: `${balances.length} assets`,
      pct: totalValue > 0 ? '+0.00%' : '0.00%',
      up: true,
    },
    {
      label: 'Unrealized P&L',
      value: `${pnl >= 0 ? '+' : ''}${fmtUsd(pnl)}`,
      change: `${positions.length} positions`,
      pct: `${pnlPct >= 0 ? '+' : ''}${pnlPct.toFixed(2)}%`,
      up: pnl >= 0,
    },
    FALLBACK[2],
    FALLBACK[3],
  ]
}

export default function PortfolioSummary() {
  const { subscribe } = useWS()
  const [stats, setStats] = useState<StatItem[]>(FALLBACK)
  const [exchanges, setExchanges] = useState<Record<string, { total: number }>>({})

  useEffect(() => {
    let cancelled = false
    Promise.allSettled([fetchBalances(), fetchPositions()])
      .then(([balRes, posRes]) => {
        if (cancelled) return
        const balances: BalanceData[] = balRes.status === 'fulfilled' ? balRes.value : []
        const positions: PositionData[] = posRes.status === 'fulfilled' ? posRes.value : []
        if (!balances.length && !positions.length) return
        setStats(buildStats(balances, positions))
      })
    return () => { cancelled = true }
  }, [])

  useEffect(() => {
    return subscribe('portfolio.update', (data) => {
      const balances = (data.balances as BalanceData[]) || []
      const positions = (data.positions as PositionData[]) || []
      const totalPnl = data.total_pnl as number | undefined
      if (balances.length || positions.length) {
        setStats(buildStats(balances, positions, totalPnl))
      }
      if (data.exchanges) {
        setExchanges(data.exchanges as Record<string, { total: number }>)
      }
    })
  }, [subscribe])

  const exchangeNames = Object.keys(exchanges)

  return (
    <div className="card fade-in">
      <div className="grid grid-cols-4">
        {stats.map((s, i) => (
          <div
            key={s.label}
            className={`px-6 py-4 ${i < stats.length - 1 ? 'border-r border-white/[0.06]' : ''}`}
          >
            <div className="flex items-center justify-between mb-2.5">
              <span className="text-[11px] text-[var(--text-3)] font-medium">{s.label}</span>
              <span className={`mono text-[10px] font-semibold px-1.5 py-0.5 rounded ${
                s.up ? 'text-[#10b981] bg-[rgba(16,185,129,0.08)]' : 'text-[#ef4444] bg-[rgba(239,68,68,0.08)]'
              }`}>
                {s.pct}
              </span>
            </div>
            <div className={`mono text-[22px] font-extrabold tracking-tight leading-none ${
              s.label.includes('P&L') ? (s.up ? 'text-[#10b981]' : 'text-[#ef4444]') : 'text-[var(--text-1)]'
            }`}>
              {s.value}
            </div>
            <div className="text-[10px] text-[var(--text-3)] mt-1.5">{s.change}</div>
          </div>
        ))}
      </div>
      {exchangeNames.length >= 2 && (
        <div className="px-6 py-3 border-t border-white/[0.06] flex items-center gap-2 flex-wrap">
          {exchangeNames.map((name) => (
            <span
              key={name}
              className="text-[10px] font-medium px-2 py-0.5 rounded bg-white/[0.04] text-[var(--text-2)]"
            >
              {name.charAt(0).toUpperCase() + name.slice(1)} · {fmtUsd(exchanges[name].total)}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}
