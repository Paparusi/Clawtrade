import { useState, useCallback } from 'react'

// ── Types ──────────────────────────────────────────────────────────────

export interface ReplayEvent {
  timestamp: number
  type: 'trade.placed' | 'trade.filled' | 'risk.alert' | 'ai.response' | 'price.update'
  data: Record<string, unknown>
}

export interface ReplaySession {
  id: string
  startTime: number
  endTime: number
  events: ReplayEvent[]
}

interface PlaybackState {
  playing: boolean
  speed: number
  currentTime: number
  currentIndex: number
}

// ── Helpers ────────────────────────────────────────────────────────────

const SPEED_OPTIONS = [1, 2, 5, 10] as const

function formatTime(ms: number): string {
  const totalSeconds = Math.floor(ms / 1000)
  const h = Math.floor(totalSeconds / 3600)
  const m = Math.floor((totalSeconds % 3600) / 60)
  const s = totalSeconds % 60
  return [h, m, s].map((v) => String(v).padStart(2, '0')).join(':')
}

function eventColor(type: ReplayEvent['type']): string {
  switch (type) {
    case 'trade.placed':
      return 'bg-green-500'
    case 'trade.filled':
      return 'bg-red-500'
    case 'risk.alert':
      return 'bg-yellow-500'
    case 'ai.response':
      return 'bg-blue-500'
    case 'price.update':
      return 'bg-gray-400'
  }
}

function eventLabel(type: ReplayEvent['type']): string {
  switch (type) {
    case 'trade.placed':
      return 'Trade Placed'
    case 'trade.filled':
      return 'Trade Filled'
    case 'risk.alert':
      return 'Risk Alert'
    case 'ai.response':
      return 'AI Response'
    case 'price.update':
      return 'Price Update'
  }
}

// ── ReplayControls ─────────────────────────────────────────────────────

interface ReplayControlsProps {
  state: PlaybackState
  session: ReplaySession
  onTogglePlay: () => void
  onSpeedChange: (speed: number) => void
  onStepForward: () => void
  onStepBack: () => void
  onSeek: (time: number) => void
}

export function ReplayControls({
  state,
  session,
  onTogglePlay,
  onSpeedChange,
  onStepForward,
  onStepBack,
  onSeek,
}: ReplayControlsProps) {
  const duration = session.endTime - session.startTime
  const progress = duration > 0 ? ((state.currentTime - session.startTime) / duration) * 100 : 0

  const handleSeek = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const pct = Number(e.target.value)
      const time = session.startTime + (pct / 100) * duration
      onSeek(time)
    },
    [session.startTime, duration, onSeek],
  )

  return (
    <div className="flex flex-col gap-3 rounded-lg border border-zinc-700 bg-zinc-900 p-4">
      {/* Seek slider */}
      <input
        type="range"
        min={0}
        max={100}
        step={0.1}
        value={progress}
        onChange={handleSeek}
        className="w-full accent-blue-500"
      />

      <div className="flex items-center justify-between">
        {/* Playback buttons */}
        <div className="flex items-center gap-2">
          <button
            onClick={onStepBack}
            className="rounded bg-zinc-800 px-3 py-1 text-sm text-zinc-300 hover:bg-zinc-700"
          >
            &#9198; Prev
          </button>
          <button
            onClick={onTogglePlay}
            className="rounded bg-blue-600 px-4 py-1 text-sm font-medium text-white hover:bg-blue-500"
          >
            {state.playing ? '⏸ Pause' : '▶ Play'}
          </button>
          <button
            onClick={onStepForward}
            className="rounded bg-zinc-800 px-3 py-1 text-sm text-zinc-300 hover:bg-zinc-700"
          >
            Next &#9197;
          </button>
        </div>

        {/* Current time */}
        <span className="font-mono text-sm text-zinc-400">
          {formatTime(state.currentTime - session.startTime)} / {formatTime(duration)}
        </span>

        {/* Speed selector */}
        <div className="flex items-center gap-1">
          {SPEED_OPTIONS.map((s) => (
            <button
              key={s}
              onClick={() => onSpeedChange(s)}
              className={`rounded px-2 py-1 text-xs font-medium ${
                state.speed === s
                  ? 'bg-blue-600 text-white'
                  : 'bg-zinc-800 text-zinc-400 hover:bg-zinc-700'
              }`}
            >
              {s}x
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

// ── ReplayTimeline ─────────────────────────────────────────────────────

interface ReplayTimelineProps {
  session: ReplaySession
  currentTime: number
}

export function ReplayTimeline({ session, currentTime }: ReplayTimelineProps) {
  const duration = session.endTime - session.startTime

  return (
    <div className="rounded-lg border border-zinc-700 bg-zinc-900 p-4">
      <h3 className="mb-2 text-sm font-semibold text-zinc-300">Timeline</h3>
      <div className="relative h-8 w-full rounded bg-zinc-800">
        {/* Playhead */}
        {duration > 0 && (
          <div
            className="absolute top-0 h-full w-0.5 bg-white"
            style={{ left: `${((currentTime - session.startTime) / duration) * 100}%` }}
          />
        )}

        {/* Event markers */}
        {session.events.map((event, i) => {
          const pct = duration > 0 ? ((event.timestamp - session.startTime) / duration) * 100 : 0
          return (
            <div
              key={i}
              title={`${eventLabel(event.type)} @ ${formatTime(event.timestamp - session.startTime)}`}
              className={`absolute top-1/2 h-3 w-3 -translate-x-1/2 -translate-y-1/2 rounded-full ${eventColor(event.type)}`}
              style={{ left: `${pct}%` }}
            />
          )
        })}
      </div>

      {/* Legend */}
      <div className="mt-2 flex flex-wrap gap-3 text-xs text-zinc-400">
        <span className="flex items-center gap-1">
          <span className="inline-block h-2 w-2 rounded-full bg-green-500" /> Trade Placed
        </span>
        <span className="flex items-center gap-1">
          <span className="inline-block h-2 w-2 rounded-full bg-red-500" /> Trade Filled
        </span>
        <span className="flex items-center gap-1">
          <span className="inline-block h-2 w-2 rounded-full bg-yellow-500" /> Risk Alert
        </span>
        <span className="flex items-center gap-1">
          <span className="inline-block h-2 w-2 rounded-full bg-blue-500" /> AI Response
        </span>
        <span className="flex items-center gap-1">
          <span className="inline-block h-2 w-2 rounded-full bg-gray-400" /> Price Update
        </span>
      </div>
    </div>
  )
}

// ── EventLog ───────────────────────────────────────────────────────────

interface EventLogProps {
  events: ReplayEvent[]
  sessionStartTime: number
}

export function EventLog({ events, sessionStartTime }: EventLogProps) {
  return (
    <div className="rounded-lg border border-zinc-700 bg-zinc-900 p-4">
      <h3 className="mb-2 text-sm font-semibold text-zinc-300">Event Log</h3>
      <div className="max-h-64 overflow-y-auto">
        {events.length === 0 ? (
          <p className="text-sm text-zinc-500">No events at current position.</p>
        ) : (
          <ul className="space-y-1">
            {events.map((event, i) => (
              <li
                key={i}
                className="flex items-start gap-3 rounded px-2 py-1 text-sm hover:bg-zinc-800"
              >
                <span className="shrink-0 font-mono text-zinc-500">
                  {formatTime(event.timestamp - sessionStartTime)}
                </span>
                <span
                  className={`shrink-0 rounded px-1.5 py-0.5 text-xs font-medium ${eventColor(event.type)} text-white`}
                >
                  {eventLabel(event.type)}
                </span>
                <span className="truncate text-zinc-300">
                  {JSON.stringify(event.data)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

// ── ReplayViewer (main) ────────────────────────────────────────────────

interface ReplayViewerProps {
  session: ReplaySession
}

export default function ReplayViewer({ session }: ReplayViewerProps) {
  const [state, setState] = useState<PlaybackState>({
    playing: false,
    speed: 1,
    currentTime: session.startTime,
    currentIndex: 0,
  })

  const togglePlay = useCallback(() => {
    setState((prev) => ({ ...prev, playing: !prev.playing }))
  }, [])

  const changeSpeed = useCallback((speed: number) => {
    setState((prev) => ({ ...prev, speed }))
  }, [])

  const stepForward = useCallback(() => {
    setState((prev) => {
      const nextIndex = Math.min(prev.currentIndex + 1, session.events.length - 1)
      return {
        ...prev,
        playing: false,
        currentIndex: nextIndex,
        currentTime: session.events[nextIndex]?.timestamp ?? prev.currentTime,
      }
    })
  }, [session.events])

  const stepBack = useCallback(() => {
    setState((prev) => {
      const prevIndex = Math.max(prev.currentIndex - 1, 0)
      return {
        ...prev,
        playing: false,
        currentIndex: prevIndex,
        currentTime: session.events[prevIndex]?.timestamp ?? prev.currentTime,
      }
    })
  }, [session.events])

  const seek = useCallback(
    (time: number) => {
      const clamped = Math.max(session.startTime, Math.min(time, session.endTime))
      let idx = 0
      for (let i = 0; i < session.events.length; i++) {
        if (session.events[i].timestamp <= clamped) {
          idx = i
        } else {
          break
        }
      }
      setState((prev) => ({
        ...prev,
        currentTime: clamped,
        currentIndex: idx,
      }))
    },
    [session.startTime, session.endTime, session.events],
  )

  const visibleEvents = session.events.filter((e) => e.timestamp <= state.currentTime)

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-bold text-white">Session Replay</h2>
        <span className="text-sm text-zinc-400">Session {session.id}</span>
      </div>

      <ReplayControls
        state={state}
        session={session}
        onTogglePlay={togglePlay}
        onSpeedChange={changeSpeed}
        onStepForward={stepForward}
        onStepBack={stepBack}
        onSeek={seek}
      />

      <ReplayTimeline session={session} currentTime={state.currentTime} />

      <EventLog events={visibleEvents} sessionStartTime={session.startTime} />
    </div>
  )
}
