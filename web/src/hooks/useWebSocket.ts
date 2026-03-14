import { useEffect, useRef, useCallback, useState } from 'react'

type EventCallback = (data: Record<string, unknown>) => void

interface UseWebSocketReturn {
  subscribe: (eventType: string, callback: EventCallback) => () => void
  connected: boolean
  lastEventTime: number | null
}

const DEV_BASE = window.location.port === '5173' ? 'http://127.0.0.1:8899' : ''

export function useWebSocket(): UseWebSocketReturn {
  const wsRef = useRef<WebSocket | null>(null)
  const listenersRef = useRef<Map<string, Set<EventCallback>>>(new Map())
  const subscribedRef = useRef<Set<string>>(new Set())
  const [connected, setConnected] = useState(false)
  const [lastEventTime, setLastEventTime] = useState<number | null>(null)
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>()
  const reconnectDelay = useRef(1000)

  const connect = useCallback(() => {
    const wsBase = DEV_BASE || `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}`
    const wsUrl = `${wsBase.replace(/^http/, 'ws')}/ws`
    const ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      setConnected(true)
      reconnectDelay.current = 1000
      for (const eventType of subscribedRef.current) {
        ws.send(JSON.stringify({ type: 'subscribe', data: eventType }))
      }
    }

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        setLastEventTime(Date.now())
        const callbacks = listenersRef.current.get(msg.type)
        if (callbacks) {
          for (const cb of callbacks) {
            cb(msg.data)
          }
        }
      } catch {
        // ignore non-JSON
      }
    }

    ws.onclose = () => {
      setConnected(false)
      reconnectTimer.current = setTimeout(() => {
        reconnectDelay.current = Math.min(reconnectDelay.current * 2, 30000)
        connect()
      }, reconnectDelay.current)
    }

    ws.onerror = () => ws.close()

    wsRef.current = ws
  }, [])

  useEffect(() => {
    connect()
    return () => {
      clearTimeout(reconnectTimer.current)
      wsRef.current?.close()
    }
  }, [connect])

  const subscribe = useCallback((eventType: string, callback: EventCallback) => {
    if (!listenersRef.current.has(eventType)) {
      listenersRef.current.set(eventType, new Set())
    }
    listenersRef.current.get(eventType)!.add(callback)

    if (!subscribedRef.current.has(eventType)) {
      subscribedRef.current.add(eventType)
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({ type: 'subscribe', data: eventType }))
      }
    }

    return () => {
      const set = listenersRef.current.get(eventType)
      if (set) {
        set.delete(callback)
        if (set.size === 0) {
          listenersRef.current.delete(eventType)
        }
      }
    }
  }, [])

  return { subscribe, connected, lastEventTime }
}
