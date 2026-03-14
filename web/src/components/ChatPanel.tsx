import { useState, useRef, useEffect } from 'react'

interface Message {
  role: 'user' | 'assistant'
  content: string
  time: string
}

function now() {
  return new Date().toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })
}

export default function ChatPanel() {
  const [messages, setMessages] = useState<Message[]>([
    { role: 'assistant', content: 'Hey! I\'m your AI trading copilot. Ask me about markets, strategies, or risk management.', time: now() },
  ])
  const [input, setInput] = useState('')
  const [typing, setTyping] = useState(false)
  const endRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, typing])

  const send = () => {
    if (!input.trim()) return
    setMessages(p => [...p, { role: 'user', content: input, time: now() }])
    const q = input
    setInput('')
    setTyping(true)
    setTimeout(() => {
      setTyping(false)
      setMessages(p => [...p, { role: 'assistant', content: respond(q), time: now() }])
    }, 1000 + Math.random() * 800)
  }

  return (
    <div className="flex flex-col h-full glass overflow-hidden">
      {/* Header */}
      <div className="flex items-center gap-3 px-5 py-3 border-b border-white/[0.04]">
        <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-[#4f8fff] to-[#00e5ff] p-[1px]">
          <div className="w-full h-full rounded-[11px] bg-[#0b1120] flex items-center justify-center">
            <span className="text-[11px]">🤖</span>
          </div>
        </div>
        <div className="flex-1">
          <div className="text-[13px] font-semibold text-white">Clawtrade AI</div>
          <div className="flex items-center gap-1.5">
            <div className="w-1.5 h-1.5 rounded-full bg-[#00dc82] pulse-glow" style={{ color: '#00dc82' }} />
            <span className="text-[10px] text-slate-500">GPT-4 · Ready</span>
          </div>
        </div>
        <div className="text-[9px] text-slate-600 bg-white/[0.03] px-2 py-1 rounded-md">
          137 tokens
        </div>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-4 py-4 space-y-4">
        {messages.map((m, i) => (
          <div key={i} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'} slide-in`}>
            <div className="max-w-[88%]">
              <div className={`px-4 py-3 text-[13px] leading-relaxed ${
                m.role === 'user'
                  ? 'bg-gradient-to-r from-[#4f8fff] to-[#4f8fff]/80 text-white rounded-2xl rounded-br-lg'
                  : 'bg-white/[0.04] border border-white/[0.06] text-slate-200 rounded-2xl rounded-bl-lg'
              }`}>
                {m.content}
              </div>
              <div className={`text-[9px] text-slate-600 mt-1.5 px-2 ${m.role === 'user' ? 'text-right' : ''}`}>
                {m.time}
              </div>
            </div>
          </div>
        ))}
        {typing && (
          <div className="flex justify-start slide-in">
            <div className="bg-white/[0.04] border border-white/[0.06] rounded-2xl rounded-bl-lg px-4 py-3">
              <div className="flex gap-1.5">
                {[0, 1, 2].map(i => (
                  <div key={i} className="w-1.5 h-1.5 rounded-full bg-[#4f8fff]/60 animate-bounce" style={{ animationDelay: `${i * 150}ms` }} />
                ))}
              </div>
            </div>
          </div>
        )}
        <div ref={endRef} />
      </div>

      {/* Input */}
      <div className="p-3 border-t border-white/[0.04]">
        <div className="flex gap-2 items-center bg-white/[0.03] rounded-xl border border-white/[0.05] px-3 py-1 focus-within:border-[#4f8fff]/30 transition-colors">
          <input
            type="text"
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && send()}
            placeholder="Ask about markets, strategies..."
            className="flex-1 py-2 bg-transparent text-[13px] text-white placeholder-slate-600 outline-none"
          />
          <button
            onClick={send}
            disabled={!input.trim()}
            className="w-8 h-8 rounded-lg bg-[#4f8fff] hover:bg-[#4f8fff]/80 disabled:opacity-20 flex items-center justify-center transition-all shrink-0"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2.5" strokeLinecap="round">
              <path d="M5 12h14M12 5l7 7-7 7"/>
            </svg>
          </button>
        </div>
      </div>
    </div>
  )
}

function respond(q: string): string {
  const l = q.toLowerCase()
  if (l.includes('btc') || l.includes('bitcoin'))
    return 'BTC is at $70,200 (+2.48%). RSI(14) at 62 on the 4H — bullish but nearing overbought. Strong support at $68.5k with heavy bid wall. Your long is in profit — consider trailing stop at $69,200 to lock gains.'
  if (l.includes('eth'))
    return 'ETH at $3,380 (-2.03%). MACD showing bearish divergence on daily. Support at $3,300. Your ETH long is -$84 underwater. If $3,300 breaks, consider cutting losses. R:R isn\'t favorable here.'
  if (l.includes('risk') || l.includes('portfolio'))
    return 'Portfolio risk score: 6.2/10 (moderate). Total exposure: $2,063 margin across 3 positions. Max drawdown today: -2.1%. Suggestion: Your BTC position is well-placed, but ETH long has negative R:R. Consider reducing ETH exposure.'
  if (l.includes('strat'))
    return 'Current market favors momentum on 4H. RSI crossover + volume breakout strategy has shown 68% win rate in recent backtests. For your risk profile, I\'d suggest:\n\n1. Scale into winners (BTC, SOL)\n2. Cut losers fast (ETH if < $3,300)\n3. Keep position sizing < 2% per trade'
  return 'I can help with:\n• **Market analysis** — "How is BTC doing?"\n• **Risk assessment** — "What\'s my portfolio risk?"\n• **Strategy advice** — "What strategy should I use?"\n• **Position management** — "Should I close my ETH?"\n\nWhat would you like to explore?'
}
