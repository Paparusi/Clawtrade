import { useEffect, useRef, useState } from 'react'

const SYMBOLS: { label: string; tv: string }[] = [
  { label: 'BTC/USDT', tv: 'BINANCE:BTCUSDT' },
  { label: 'ETH/USDT', tv: 'BINANCE:ETHUSDT' },
  { label: 'SOL/USDT', tv: 'BINANCE:SOLUSDT' },
  { label: 'BNB/USDT', tv: 'BINANCE:BNBUSDT' },
  { label: 'EUR/USD', tv: 'FX:EURUSD' },
  { label: 'XAU/USD', tv: 'TVC:GOLD' },
  { label: 'AAPL', tv: 'NASDAQ:AAPL' },
  { label: 'NVDA', tv: 'NASDAQ:NVDA' },
]

export default function TradingViewChart() {
  const containerRef = useRef<HTMLDivElement>(null)
  const [symbol, setSymbol] = useState(SYMBOLS[0])

  useEffect(() => {
    if (!containerRef.current) return
    containerRef.current.innerHTML = ''

    const script = document.createElement('script')
    script.src = 'https://s3.tradingview.com/external-embedding/embed-widget-advanced-chart.js'
    script.type = 'text/javascript'
    script.async = true
    script.innerHTML = JSON.stringify({
      autosize: true,
      symbol: symbol.tv,
      interval: '60',
      timezone: 'Etc/UTC',
      theme: 'dark',
      style: '1',
      locale: 'en',
      backgroundColor: 'rgba(9, 9, 11, 1)',
      gridColor: 'rgba(255, 255, 255, 0.03)',
      hide_top_toolbar: false,
      hide_legend: false,
      save_image: false,
      calendar: false,
      hide_volume: false,
      support_host: 'https://www.tradingview.com',
    })

    const wrapper = document.createElement('div')
    wrapper.className = 'tradingview-widget-container__widget'
    wrapper.style.height = '100%'
    wrapper.style.width = '100%'

    containerRef.current.appendChild(wrapper)
    containerRef.current.appendChild(script)
  }, [symbol])

  return (
    <div className="card fade-in flex flex-col h-full" style={{ animationDelay: '100ms' }}>
      <div className="flex items-center gap-2 px-4 py-2.5 border-b border-white/[0.06]">
        <span className="text-xs font-bold text-[var(--text-1)]">Chart</span>
        <div className="flex gap-1 ml-2">
          {SYMBOLS.map(s => (
            <button
              key={s.tv}
              onClick={() => setSymbol(s)}
              className={`px-2 py-0.5 rounded text-[10px] font-semibold border-none cursor-pointer transition-all ${
                symbol.tv === s.tv
                  ? 'bg-[rgba(99,102,241,0.1)] text-[#818cf8]'
                  : 'bg-transparent text-[var(--text-3)] hover:text-[var(--text-2)]'
              }`}
            >
              {s.label}
            </button>
          ))}
        </div>
      </div>
      <div
        ref={containerRef}
        className="tradingview-widget-container flex-1 min-h-0"
        style={{ overflow: 'hidden' }}
      />
    </div>
  )
}
