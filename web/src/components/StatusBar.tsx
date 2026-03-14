import { useState, useEffect } from 'react'
import { useWS } from '../hooks/WebSocketProvider'

export default function StatusBar() {
  const { connected, lastEventTime } = useWS()
  const [, setTick] = useState(0)

  useEffect(() => {
    const timer = setInterval(() => setTick(t => t + 1), 5000)
    return () => clearInterval(timer)
  }, [])

  const lastUpdate = lastEventTime
    ? `${Math.floor((Date.now() - lastEventTime) / 1000)}s ago`
    : 'No events yet'

  return (
    <div className="flex items-center justify-between px-4 py-1.5 border-t border-white/[0.06] bg-[var(--bg-1)]">
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-1.5">
          <div
            className={`w-1.5 h-1.5 rounded-full ${connected ? 'bg-[#10b981] pulse-dot' : 'bg-[#ef4444]'}`}
          />
          <span className="text-[10px] text-[var(--text-3)]">
            {connected ? 'WebSocket Connected' : 'Disconnected'}
          </span>
        </div>
      </div>
      <span className="text-[10px] text-[var(--text-3)]">
        Last update: {lastUpdate}
      </span>
    </div>
  )
}
