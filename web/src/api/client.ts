const DEV_BASE = window.location.port === '5173' ? 'http://127.0.0.1:8899' : ''
const API_BASE = `${DEV_BASE}/api/v1`

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`)
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error || `HTTP ${res.status}`)
  }
  return res.json()
}

// ─── System ────────────────────────────────────────────────────────

export async function fetchHealth() {
  return get<{ status: string; version: string }>('/system/health')
}

export async function fetchVersion() {
  return get<{ version: string }>('/system/version')
}

// ─── Market Data ───────────────────────────────────────────────────

export interface PriceData {
  symbol: string
  bid: number
  ask: number
  last: number
  volume_24h: number
  timestamp: string
}

export async function fetchPrice(symbol: string, exchange = 'binance') {
  return get<PriceData>(`/price?symbol=${encodeURIComponent(symbol)}&exchange=${exchange}`)
}

export interface CandleData {
  open: number
  high: number
  low: number
  close: number
  volume: number
  timestamp: string
}

export async function fetchCandles(symbol: string, timeframe = '1h', limit = 100, exchange = 'binance') {
  return get<CandleData[]>(`/candles?symbol=${encodeURIComponent(symbol)}&timeframe=${timeframe}&limit=${limit}&exchange=${exchange}`)
}

// ─── Account ───────────────────────────────────────────────────────

export interface BalanceData {
  asset: string
  free: number
  locked: number
  total: number
}

export async function fetchBalances(exchange = 'binance') {
  return get<BalanceData[]>(`/balances?exchange=${exchange}`)
}

export interface PositionData {
  symbol: string
  side: string
  size: number
  entry_price: number
  current_price: number
  pnl: number
  exchange: string
  opened_at: string
}

export async function fetchPositions(exchange = 'binance') {
  return get<PositionData[]>(`/positions?exchange=${exchange}`)
}

export interface ExchangeInfo {
  name: string
  connected: boolean
  caps: {
    name: string
    websocket: boolean
    margin: boolean
    futures: boolean
    order_types: string[]
  }
}

export async function fetchExchanges() {
  return get<ExchangeInfo[]>('/exchanges')
}

export interface PortfolioData {
  balances: BalanceData[]
  positions: PositionData[]
  total_pnl: number
  exchanges: Record<string, {
    total: number
    balances: BalanceData[]
    positions: PositionData[]
  }>
}

export async function fetchPortfolio() {
  return get<PortfolioData>('/portfolio')
}

// ─── WebSocket ─────────────────────────────────────────────────────

export interface WSMessage {
  type: string
  data: Record<string, unknown>
}

export function connectWebSocket(onMessage: (msg: WSMessage) => void) {
  const wsBase = DEV_BASE || `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}`
  const wsUrl = `${wsBase.replace(/^http/, 'ws')}/ws`
  const ws = new WebSocket(wsUrl)

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      onMessage(data)
    } catch {
      // ignore non-JSON
    }
  }

  return ws
}
