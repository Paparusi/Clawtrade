import { createContext, useContext, type ReactNode } from 'react'
import { useWebSocket } from './useWebSocket'

type EventCallback = (data: Record<string, unknown>) => void

interface WebSocketContextValue {
  subscribe: (eventType: string, callback: EventCallback) => () => void
  connected: boolean
  lastEventTime: number | null
}

const WebSocketContext = createContext<WebSocketContextValue | null>(null)

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const ws = useWebSocket()
  return (
    <WebSocketContext.Provider value={ws}>
      {children}
    </WebSocketContext.Provider>
  )
}

export function useWS(): WebSocketContextValue {
  const ctx = useContext(WebSocketContext)
  if (!ctx) throw new Error('useWS must be used within WebSocketProvider')
  return ctx
}
