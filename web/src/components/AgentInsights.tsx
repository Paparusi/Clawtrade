import { useState, useEffect, useRef } from 'react'
import { useWS } from '../hooks/WebSocketProvider'

interface InsightEvent {
  id: number
  type: string
  source: string
  symbol: string
  summary: string
  data: Record<string, unknown>
  time: number
}

const AGENT_EVENTS = [
  'agent.analysis',
  'agent.counter',
  'agent.narrative',
  'agent.reflection',
  'agent.correlation',
]

const ICONS: Record<string, string> = {
  'agent.analysis': '\u{1F50D}',
  'agent.counter': '\u{2694}\uFE0F',
  'agent.narrative': '\u{1F4D6}',
  'agent.reflection': '\u{1FA9E}',
  'agent.correlation': '\u{1F517}',
}

const LABELS: Record<string, string> = {
  'agent.analysis': 'Analysis',
  'agent.counter': 'Counter',
  'agent.narrative': 'Narrative',
  'agent.reflection': 'Reflection',
  'agent.correlation': 'Correlation',
}

function timeAgo(ts: number): string {
  const diff = Math.floor((Date.now() - ts) / 1000)
  if (diff < 60) return `${diff}s ago`
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
  return `${Math.floor(diff / 3600)}h ago`
}

let nextId = 0

export default function AgentInsights() {
  const { subscribe } = useWS()
  const [events, setEvents] = useState<InsightEvent[]>([])
  const [expanded, setExpanded] = useState<number | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)
  const [, setTick] = useState(0)

  useEffect(() => {
    const unsubs = AGENT_EVENTS.map(eventType =>
      subscribe(eventType, (data) => {
        const event: InsightEvent = {
          id: nextId++,
          type: eventType,
          source: (data.source as string) || 'unknown',
          symbol: (data.symbol as string) || '',
          summary: (data.summary as string) || '',
          data: (data.data as Record<string, unknown>) || {},
          time: Date.now(),
        }
        setEvents(prev => {
          const updated = [event, ...prev]
          return updated.slice(0, 50)
        })
      })
    )
    return () => unsubs.forEach(fn => fn())
  }, [subscribe])

  useEffect(() => {
    const timer = setInterval(() => setTick(t => t + 1), 30000)
    return () => clearInterval(timer)
  }, [])

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 py-2.5 border-b border-white/[0.06]">
        <span className="text-xs font-bold text-[var(--text-1)]">Agent Insights</span>
        <span className="text-[10px] text-[var(--text-3)]">{events.length} events</span>
      </div>

      <div ref={scrollRef} className="flex-1 overflow-y-auto">
        {events.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-center px-4">
            <div className="text-2xl mb-2 opacity-30">{'\u{1F916}'}</div>
            <p className="text-[11px] text-[var(--text-3)]">Waiting for agent insights...</p>
            <p className="text-[9px] text-[var(--text-3)] mt-1">Sub-agents will stream analysis here</p>
          </div>
        ) : (
          events.map(ev => (
            <div
              key={ev.id}
              className="px-4 py-3 border-b border-white/[0.03] cursor-pointer hover:bg-white/[0.02] transition-colors"
              onClick={() => setExpanded(expanded === ev.id ? null : ev.id)}
            >
              <div className="flex items-start gap-2">
                <span className="text-sm mt-0.5">{ICONS[ev.type] || '\u{1F4AC}'}</span>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-[10px] font-semibold text-[#818cf8]">
                      {LABELS[ev.type] || ev.type}
                    </span>
                    {ev.symbol && (
                      <span className="text-[9px] font-medium text-[var(--text-2)]">{ev.symbol}</span>
                    )}
                    <span className="text-[9px] text-[var(--text-3)] ml-auto shrink-0">
                      {timeAgo(ev.time)}
                    </span>
                  </div>
                  <p className="text-[11px] text-[var(--text-2)] mt-1 leading-relaxed">
                    {ev.summary}
                  </p>
                  {expanded === ev.id && (
                    <pre className="text-[9px] text-[var(--text-3)] mt-2 p-2 rounded bg-[var(--bg-0)] overflow-x-auto whitespace-pre-wrap">
                      {JSON.stringify(ev.data, null, 2)}
                    </pre>
                  )}
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
