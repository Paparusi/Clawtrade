import { useState, useEffect } from 'react'
import { useWS } from '../hooks/WebSocketProvider'

interface Toast {
  id: number
  symbol: string
  side: string
  price: number
  size: number
  time: number
}

let toastId = 0

export default function TradeToast() {
  const { subscribe } = useWS()
  const [toasts, setToasts] = useState<Toast[]>([])

  useEffect(() => {
    return subscribe('trade.executed', (data) => {
      const toast: Toast = {
        id: toastId++,
        symbol: (data.symbol as string) || '',
        side: (data.side as string) || '',
        price: (data.price as number) || 0,
        size: (data.size as number) || 0,
        time: Date.now(),
      }
      setToasts(prev => [...prev, toast])

      setTimeout(() => {
        setToasts(prev => prev.filter(t => t.id !== toast.id))
      }, 5000)
    })
  }, [subscribe])

  if (!toasts.length) return null

  return (
    <div className="fixed top-4 right-4 z-50 flex flex-col gap-2">
      {toasts.map(t => {
        const isBuy = t.side.toLowerCase() === 'buy'
        return (
          <div
            key={t.id}
            className="fade-in rounded-lg border px-4 py-3 shadow-lg min-w-[280px]"
            style={{
              background: 'var(--bg-1)',
              borderColor: isBuy ? 'rgba(16,185,129,0.2)' : 'rgba(239,68,68,0.2)',
            }}
          >
            <div className="flex items-center gap-3">
              <div
                className="w-8 h-8 rounded-lg flex items-center justify-center text-xs font-bold"
                style={{
                  background: isBuy ? 'rgba(16,185,129,0.1)' : 'rgba(239,68,68,0.1)',
                  color: isBuy ? '#10b981' : '#ef4444',
                }}
              >
                {isBuy ? 'B' : 'S'}
              </div>
              <div>
                <div className="text-[12px] font-semibold text-[var(--text-1)]">
                  {t.side} {t.symbol}
                </div>
                <div className="mono text-[11px] text-[var(--text-2)]">
                  {t.size} @ ${t.price.toLocaleString()}
                </div>
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )
}
