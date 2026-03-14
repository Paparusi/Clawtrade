const exchanges = [
  { name: 'Binance', type: 'CEX', status: 'connected' as const, pairs: 342 },
  { name: 'Bybit', type: 'CEX', status: 'connected' as const, pairs: 286 },
  { name: 'MetaTrader 5', type: 'Broker', status: 'connected' as const, pairs: 78 },
  { name: 'Interactive Brokers', type: 'Broker', status: 'ready' as const, pairs: 1200 },
  { name: 'Hyperliquid', type: 'DEX', status: 'connected' as const, pairs: 45 },
  { name: 'Uniswap', type: 'DEX', status: 'ready' as const, pairs: 500 },
]

export default function ExchangeStatus() {
  const on = exchanges.filter(e => e.status === 'connected').length
  return (
    <div className="card fade-in" style={{ animationDelay: '250ms' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 16px', borderBottom: '1px solid var(--border)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontSize: 12, fontWeight: 700, color: 'var(--text-1)' }}>Exchanges</span>
          <span style={{ fontSize: 10, color: '#10b981', fontWeight: 600 }}>{on}/{exchanges.length}</span>
        </div>
      </div>
      <div style={{ padding: 8, display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 4 }}>
        {exchanges.map(ex => (
          <div
            key={ex.name}
            style={{
              display: 'flex', alignItems: 'center', gap: 8,
              padding: '8px 10px', borderRadius: 8, cursor: 'pointer',
              transition: 'background 0.15s',
            }}
            onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255,255,255,0.02)'}
            onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}
          >
            <div style={{
              width: 6, height: 6, borderRadius: '50%',
              background: ex.status === 'connected' ? '#10b981' : 'var(--bg-3)',
            }} />
            <div>
              <div style={{ fontSize: 11, fontWeight: 500, color: 'var(--text-1)' }}>{ex.name}</div>
              <div style={{ fontSize: 9, color: 'var(--text-3)' }}>{ex.type} · {ex.pairs} pairs</div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
