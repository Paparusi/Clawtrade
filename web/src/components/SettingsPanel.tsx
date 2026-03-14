import { useState } from 'react'

type Section = 'exchanges' | 'risk' | 'agent' | 'notifications' | 'general'

const sections: { id: Section; label: string; icon: JSX.Element }[] = [
  {
    id: 'exchanges', label: 'Exchanges',
    icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M2 12h5l2-9 6 18 2-9h5"/></svg>,
  },
  {
    id: 'risk', label: 'Risk Management',
    icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>,
  },
  {
    id: 'agent', label: 'Agent Config',
    icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M12 2a8 8 0 0 1 8 8v1a8 8 0 0 1-16 0v-1a8 8 0 0 1 8-8z"/><circle cx="9" cy="9.5" r="1"/><circle cx="15" cy="9.5" r="1"/><path d="M9 14c1.5 1.2 4.5 1.2 6 0"/></svg>,
  },
  {
    id: 'notifications', label: 'Notifications',
    icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>,
  },
  {
    id: 'general', label: 'General',
    icon: <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>,
  },
]

interface ExchangeConfig {
  name: string
  type: string
  connected: boolean
  fields: Record<string, string>
}

const defaultExchanges: ExchangeConfig[] = [
  { name: 'Binance', type: 'CEX', connected: true, fields: { apiKey: '****...7f3a', apiSecret: '****...9b2c', keyType: 'ed25519', env: 'production', ipWhitelist: '' } },
  { name: 'Bybit', type: 'CEX', connected: true, fields: { apiKey: '****...4d1e', apiSecret: '****...8a5f', keyType: 'hmac', env: 'production' } },
  { name: 'MetaTrader 5', type: 'Broker', connected: false, fields: { login: '', password: '', server: '', accountType: 'demo' } },
  { name: 'Interactive Brokers', type: 'Broker', connected: false, fields: { username: '', accountId: '', connection: 'gateway', host: '127.0.0.1', port: '4002', accountType: 'paper', clientId: '1' } },
  { name: 'Hyperliquid', type: 'DEX', connected: true, fields: { walletAddress: '****...2e8b', privateKey: '****...1c4d', useApiWallet: 'false', apiWalletAddress: '', apiWalletKey: '', env: 'mainnet' } },
  { name: 'Uniswap', type: 'DEX', connected: false, fields: { walletAddress: '', privateKey: '', chain: 'ethereum', rpcUrl: '', slippage: '0.5', gasMode: 'standard' } },
]

// Reusable components
function Toggle({ value, onChange }: { value: boolean; onChange: (v: boolean) => void }) {
  return (
    <button
      onClick={() => onChange(!value)}
      style={{
        width: 40, height: 22, borderRadius: 11, border: 'none', cursor: 'pointer',
        background: value ? '#6366f1' : 'var(--bg-3)', transition: 'background 0.2s',
        position: 'relative', flexShrink: 0,
      }}
    >
      <div style={{
        width: 16, height: 16, borderRadius: '50%', background: 'white',
        position: 'absolute', top: 3, left: value ? 21 : 3, transition: 'left 0.2s',
      }} />
    </button>
  )
}

function FieldRow({ label, desc, children }: { label: string; desc?: string; children: React.ReactNode }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '14px 0', borderBottom: '1px solid var(--border)' }}>
      <div style={{ flex: 1 }}>
        <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-1)' }}>{label}</div>
        {desc && <div style={{ fontSize: 11, color: 'var(--text-3)', marginTop: 2 }}>{desc}</div>}
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>{children}</div>
    </div>
  )
}

function NumberInput({ value, onChange, min, max, step, unit }: { value: number; onChange: (v: number) => void; min?: number; max?: number; step?: number; unit?: string }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
      <input
        type="number"
        value={value}
        onChange={e => onChange(Number(e.target.value))}
        min={min} max={max} step={step}
        style={{
          width: 80, padding: '6px 10px', borderRadius: 6, border: '1px solid var(--border)',
          background: 'var(--bg-2)', color: 'var(--text-1)', fontSize: 12, fontFamily: "'JetBrains Mono', monospace",
          outline: 'none', textAlign: 'right',
        }}
      />
      {unit && <span style={{ fontSize: 11, color: 'var(--text-3)', minWidth: 20 }}>{unit}</span>}
    </div>
  )
}

function SelectInput({ value, options, onChange }: { value: string; options: { value: string; label: string }[]; onChange: (v: string) => void }) {
  return (
    <select
      value={value}
      onChange={e => onChange(e.target.value)}
      style={{
        padding: '6px 10px', borderRadius: 6, border: '1px solid var(--border)',
        background: 'var(--bg-2)', color: 'var(--text-1)', fontSize: 12, outline: 'none',
        cursor: 'pointer',
      }}
    >
      {options.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
    </select>
  )
}

function TextInput({ value, onChange, placeholder, type = 'text' }: { value: string; onChange: (v: string) => void; placeholder?: string; type?: string }) {
  return (
    <input
      type={type}
      value={value}
      onChange={e => onChange(e.target.value)}
      placeholder={placeholder}
      style={{
        width: '100%', padding: '8px 12px', borderRadius: 6, border: '1px solid var(--border)',
        background: 'var(--bg-2)', color: 'var(--text-1)', fontSize: 12, outline: 'none',
      }}
    />
  )
}

function SectionTitle({ title, desc }: { title: string; desc?: string }) {
  return (
    <div style={{ marginBottom: 20 }}>
      <h2 style={{ fontSize: 18, fontWeight: 700, color: 'var(--text-1)', marginBottom: 4 }}>{title}</h2>
      {desc && <p style={{ fontSize: 12, color: 'var(--text-3)', lineHeight: 1.5 }}>{desc}</p>}
    </div>
  )
}

function SaveButton({ onClick, saved }: { onClick: () => void; saved: boolean }) {
  return (
    <button
      onClick={onClick}
      style={{
        padding: '8px 24px', borderRadius: 8, border: 'none', cursor: 'pointer',
        background: saved ? 'rgba(16,185,129,0.1)' : '#6366f1',
        color: saved ? '#10b981' : 'white',
        fontSize: 13, fontWeight: 600, transition: 'all 0.2s',
      }}
    >
      {saved ? 'Saved' : 'Save Changes'}
    </button>
  )
}

function FormLabel({ children }: { children: React.ReactNode }) {
  return <label style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-3)', display: 'block', marginBottom: 4 }}>{children}</label>
}

function FormHint({ children }: { children: React.ReactNode }) {
  return <div style={{ fontSize: 10, color: 'var(--text-3)', marginTop: 4, opacity: 0.7 }}>{children}</div>
}

function FormRow({ children }: { children: React.ReactNode }) {
  return <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>{children}</div>
}

// Per-exchange form renderers
function BinanceForm({ fields, onChange }: { fields: Record<string, string>; onChange: (k: string, v: string) => void }) {
  return (
    <>
      <FormRow>
        <div>
          <FormLabel>API Key</FormLabel>
          <TextInput value={fields.apiKey} onChange={v => onChange('apiKey', v)} placeholder="Enter Binance API key" />
        </div>
        <div>
          <FormLabel>API Secret</FormLabel>
          <TextInput value={fields.apiSecret} onChange={v => onChange('apiSecret', v)} placeholder="Enter API secret" type="password" />
        </div>
      </FormRow>
      <FormRow>
        <div>
          <FormLabel>Signature Algorithm</FormLabel>
          <select value={fields.keyType} onChange={e => onChange('keyType', e.target.value)} style={{
            width: '100%', padding: '8px 12px', borderRadius: 6, border: '1px solid var(--border)',
            background: 'var(--bg-2)', color: 'var(--text-1)', fontSize: 12, outline: 'none', cursor: 'pointer',
          }}>
            <option value="ed25519">Ed25519 (Recommended)</option>
            <option value="hmac">HMAC-SHA256</option>
            <option value="rsa">RSA</option>
          </select>
          <FormHint>Ed25519 offers best performance & security</FormHint>
        </div>
        <div>
          <FormLabel>Environment</FormLabel>
          <select value={fields.env} onChange={e => onChange('env', e.target.value)} style={{
            width: '100%', padding: '8px 12px', borderRadius: 6, border: '1px solid var(--border)',
            background: 'var(--bg-2)', color: 'var(--text-1)', fontSize: 12, outline: 'none', cursor: 'pointer',
          }}>
            <option value="production">Production</option>
            <option value="testnet">Testnet</option>
          </select>
        </div>
      </FormRow>
      <div>
        <FormLabel>IP Whitelist (optional)</FormLabel>
        <TextInput value={fields.ipWhitelist} onChange={v => onChange('ipWhitelist', v)} placeholder="e.g. 203.0.113.1, 198.51.100.2" />
        <FormHint>Keys without IP whitelist expire after 30 days. Comma-separated IPs.</FormHint>
      </div>
    </>
  )
}

function BybitForm({ fields, onChange }: { fields: Record<string, string>; onChange: (k: string, v: string) => void }) {
  return (
    <>
      <FormRow>
        <div>
          <FormLabel>API Key</FormLabel>
          <TextInput value={fields.apiKey} onChange={v => onChange('apiKey', v)} placeholder="Enter Bybit API key" />
        </div>
        <div>
          <FormLabel>API Secret</FormLabel>
          <TextInput value={fields.apiSecret} onChange={v => onChange('apiSecret', v)} placeholder="Enter API secret" type="password" />
        </div>
      </FormRow>
      <FormRow>
        <div>
          <FormLabel>Key Type</FormLabel>
          <select value={fields.keyType} onChange={e => onChange('keyType', e.target.value)} style={{
            width: '100%', padding: '8px 12px', borderRadius: 6, border: '1px solid var(--border)',
            background: 'var(--bg-2)', color: 'var(--text-1)', fontSize: 12, outline: 'none', cursor: 'pointer',
          }}>
            <option value="hmac">System-Generated (HMAC)</option>
            <option value="rsa">Self-Generated (RSA)</option>
          </select>
        </div>
        <div>
          <FormLabel>Environment</FormLabel>
          <select value={fields.env} onChange={e => onChange('env', e.target.value)} style={{
            width: '100%', padding: '8px 12px', borderRadius: 6, border: '1px solid var(--border)',
            background: 'var(--bg-2)', color: 'var(--text-1)', fontSize: 12, outline: 'none', cursor: 'pointer',
          }}>
            <option value="production">Production</option>
            <option value="testnet">Testnet</option>
            <option value="demo">Demo Trading (Mainnet)</option>
          </select>
          <FormHint>Demo uses mainnet servers with simulated funds</FormHint>
        </div>
      </FormRow>
    </>
  )
}

function MT5Form({ fields, onChange }: { fields: Record<string, string>; onChange: (k: string, v: string) => void }) {
  return (
    <>
      <FormRow>
        <div>
          <FormLabel>Login (Account Number)</FormLabel>
          <TextInput value={fields.login} onChange={v => onChange('login', v)} placeholder="e.g. 12345678" />
        </div>
        <div>
          <FormLabel>Password</FormLabel>
          <TextInput value={fields.password} onChange={v => onChange('password', v)} placeholder="MT5 account password" type="password" />
        </div>
      </FormRow>
      <FormRow>
        <div>
          <FormLabel>Server</FormLabel>
          <TextInput value={fields.server} onChange={v => onChange('server', v)} placeholder="e.g. Forex4you-Demo, XM.COM-REAL" />
          <FormHint>Broker server name — check your MT5 terminal or broker email</FormHint>
        </div>
        <div>
          <FormLabel>Account Type</FormLabel>
          <select value={fields.accountType} onChange={e => onChange('accountType', e.target.value)} style={{
            width: '100%', padding: '8px 12px', borderRadius: 6, border: '1px solid var(--border)',
            background: 'var(--bg-2)', color: 'var(--text-1)', fontSize: 12, outline: 'none', cursor: 'pointer',
          }}>
            <option value="demo">Demo</option>
            <option value="live">Live</option>
          </select>
        </div>
      </FormRow>
    </>
  )
}

function IBKRForm({ fields, onChange }: { fields: Record<string, string>; onChange: (k: string, v: string) => void }) {
  const isGateway = fields.connection === 'gateway'
  const isPaper = fields.accountType === 'paper'
  const autoPort = isGateway ? (isPaper ? '4002' : '4001') : (isPaper ? '7497' : '7496')

  return (
    <>
      <FormRow>
        <div>
          <FormLabel>Username</FormLabel>
          <TextInput value={fields.username} onChange={v => onChange('username', v)} placeholder="IBKR username" />
        </div>
        <div>
          <FormLabel>Account ID</FormLabel>
          <TextInput value={fields.accountId} onChange={v => onChange('accountId', v)} placeholder="e.g. U12345 (auto-detected after login)" />
        </div>
      </FormRow>
      <FormRow>
        <div>
          <FormLabel>Connection Method</FormLabel>
          <select value={fields.connection} onChange={e => { onChange('connection', e.target.value); onChange('port', e.target.value === 'gateway' ? (isPaper ? '4002' : '4001') : (isPaper ? '7497' : '7496')) }} style={{
            width: '100%', padding: '8px 12px', borderRadius: 6, border: '1px solid var(--border)',
            background: 'var(--bg-2)', color: 'var(--text-1)', fontSize: 12, outline: 'none', cursor: 'pointer',
          }}>
            <option value="gateway">IB Gateway (Recommended)</option>
            <option value="tws">Trader Workstation (TWS)</option>
          </select>
          <FormHint>{isGateway ? 'Lightweight, headless — ideal for automated trading' : 'Full desktop app — enable API in Edit → Global Config → API'}</FormHint>
        </div>
        <div>
          <FormLabel>Account Type</FormLabel>
          <select value={fields.accountType} onChange={e => { onChange('accountType', e.target.value); onChange('port', isGateway ? (e.target.value === 'paper' ? '4002' : '4001') : (e.target.value === 'paper' ? '7497' : '7496')) }} style={{
            width: '100%', padding: '8px 12px', borderRadius: 6, border: '1px solid var(--border)',
            background: 'var(--bg-2)', color: 'var(--text-1)', fontSize: 12, outline: 'none', cursor: 'pointer',
          }}>
            <option value="paper">Paper Trading</option>
            <option value="live">Live Trading</option>
          </select>
        </div>
      </FormRow>
      <FormRow>
        <div>
          <FormLabel>Host</FormLabel>
          <TextInput value={fields.host} onChange={v => onChange('host', v)} placeholder="127.0.0.1" />
        </div>
        <div>
          <FormLabel>Port</FormLabel>
          <TextInput value={fields.port} onChange={v => onChange('port', v)} placeholder={autoPort} />
          <FormHint>Auto: {autoPort} for {isGateway ? 'Gateway' : 'TWS'} {isPaper ? 'Paper' : 'Live'}</FormHint>
        </div>
      </FormRow>
      <div>
        <FormLabel>Client ID</FormLabel>
        <div style={{ maxWidth: 120 }}>
          <TextInput value={fields.clientId} onChange={v => onChange('clientId', v)} placeholder="1" />
        </div>
        <FormHint>Unique per connection. Supports up to 32 simultaneous clients.</FormHint>
      </div>
    </>
  )
}

function HyperliquidForm({ fields, onChange }: { fields: Record<string, string>; onChange: (k: string, v: string) => void }) {
  const useApi = fields.useApiWallet === 'true'
  return (
    <>
      <FormRow>
        <div>
          <FormLabel>Wallet Address</FormLabel>
          <TextInput value={fields.walletAddress} onChange={v => onChange('walletAddress', v)} placeholder="0x..." />
        </div>
        <div>
          <FormLabel>Environment</FormLabel>
          <select value={fields.env} onChange={e => onChange('env', e.target.value)} style={{
            width: '100%', padding: '8px 12px', borderRadius: 6, border: '1px solid var(--border)',
            background: 'var(--bg-2)', color: 'var(--text-1)', fontSize: 12, outline: 'none', cursor: 'pointer',
          }}>
            <option value="mainnet">Mainnet</option>
            <option value="testnet">Testnet</option>
          </select>
        </div>
      </FormRow>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 0', borderBottom: '1px solid var(--border)' }}>
        <div>
          <div style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-1)' }}>Use API Wallet</div>
          <div style={{ fontSize: 10, color: 'var(--text-3)', marginTop: 2 }}>Recommended for bots — API wallets cannot withdraw funds</div>
        </div>
        <Toggle value={useApi} onChange={v => onChange('useApiWallet', String(v))} />
      </div>
      {useApi ? (
        <FormRow>
          <div>
            <FormLabel>API Wallet Address</FormLabel>
            <TextInput value={fields.apiWalletAddress} onChange={v => onChange('apiWalletAddress', v)} placeholder="0x... (generated from Hyperliquid app)" />
          </div>
          <div>
            <FormLabel>API Wallet Private Key</FormLabel>
            <TextInput value={fields.apiWalletKey} onChange={v => onChange('apiWalletKey', v)} placeholder="0x..." type="password" />
            <FormHint>Shown only once during creation. Keys expire after 180 days.</FormHint>
          </div>
        </FormRow>
      ) : (
        <div>
          <FormLabel>Private Key</FormLabel>
          <TextInput value={fields.privateKey} onChange={v => onChange('privateKey', v)} placeholder="0x..." type="password" />
          <FormHint>Direct wallet signing — has full permissions including withdrawals</FormHint>
        </div>
      )}
    </>
  )
}

function UniswapForm({ fields, onChange }: { fields: Record<string, string>; onChange: (k: string, v: string) => void }) {
  const chainRpcs: Record<string, string> = {
    ethereum: 'https://eth.rpc.bloxroute.com',
    arbitrum: 'https://arb1.arbitrum.io/rpc',
    polygon: 'https://polygon-rpc.com',
    base: 'https://mainnet.base.org',
    optimism: 'https://mainnet.optimism.io',
  }
  return (
    <>
      <FormRow>
        <div>
          <FormLabel>Wallet Address</FormLabel>
          <TextInput value={fields.walletAddress} onChange={v => onChange('walletAddress', v)} placeholder="0x..." />
        </div>
        <div>
          <FormLabel>Private Key</FormLabel>
          <TextInput value={fields.privateKey} onChange={v => onChange('privateKey', v)} placeholder="0x..." type="password" />
        </div>
      </FormRow>
      <FormRow>
        <div>
          <FormLabel>Chain</FormLabel>
          <select value={fields.chain} onChange={e => { onChange('chain', e.target.value); onChange('rpcUrl', chainRpcs[e.target.value] || '') }} style={{
            width: '100%', padding: '8px 12px', borderRadius: 6, border: '1px solid var(--border)',
            background: 'var(--bg-2)', color: 'var(--text-1)', fontSize: 12, outline: 'none', cursor: 'pointer',
          }}>
            <option value="ethereum">Ethereum (ChainID: 1)</option>
            <option value="arbitrum">Arbitrum (ChainID: 42161)</option>
            <option value="polygon">Polygon (ChainID: 137)</option>
            <option value="base">Base (ChainID: 8453)</option>
            <option value="optimism">Optimism (ChainID: 10)</option>
          </select>
        </div>
        <div>
          <FormLabel>RPC Endpoint</FormLabel>
          <TextInput value={fields.rpcUrl} onChange={v => onChange('rpcUrl', v)} placeholder={chainRpcs[fields.chain] || 'https://...'} />
          <FormHint>Public RPC or Infura/Alchemy for better reliability</FormHint>
        </div>
      </FormRow>
      <FormRow>
        <div>
          <FormLabel>Default Slippage Tolerance</FormLabel>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <input type="number" value={fields.slippage} onChange={e => onChange('slippage', e.target.value)}
              min={0.1} max={50} step={0.1}
              style={{ width: 80, padding: '8px 12px', borderRadius: 6, border: '1px solid var(--border)', background: 'var(--bg-2)', color: 'var(--text-1)', fontSize: 12, fontFamily: "'JetBrains Mono', monospace", outline: 'none', textAlign: 'right' }}
            />
            <span style={{ fontSize: 12, color: 'var(--text-3)' }}>%</span>
          </div>
          <FormHint>0.1%–5% typical. Higher for volatile tokens.</FormHint>
        </div>
        <div>
          <FormLabel>Gas Mode</FormLabel>
          <select value={fields.gasMode} onChange={e => onChange('gasMode', e.target.value)} style={{
            width: '100%', padding: '8px 12px', borderRadius: 6, border: '1px solid var(--border)',
            background: 'var(--bg-2)', color: 'var(--text-1)', fontSize: 12, outline: 'none', cursor: 'pointer',
          }}>
            <option value="standard">Standard</option>
            <option value="fast">Fast</option>
            <option value="aggressive">Aggressive</option>
          </select>
        </div>
      </FormRow>
    </>
  )
}

const exchangeFormMap: Record<string, (props: { fields: Record<string, string>; onChange: (k: string, v: string) => void }) => JSX.Element> = {
  'Binance': BinanceForm,
  'Bybit': BybitForm,
  'MetaTrader 5': MT5Form,
  'Interactive Brokers': IBKRForm,
  'Hyperliquid': HyperliquidForm,
  'Uniswap': UniswapForm,
}

// Section: Exchanges
function ExchangesSection() {
  const [exchanges, setExchanges] = useState(defaultExchanges)
  const [editing, setEditing] = useState<string | null>(null)

  const updateField = (idx: number, key: string, value: string) => {
    const next = [...exchanges]
    next[idx] = { ...next[idx], fields: { ...next[idx].fields, [key]: value } }
    setExchanges(next)
  }

  return (
    <div>
      <SectionTitle title="Exchange Connections" desc="Connect your exchange accounts. Credentials are encrypted and stored in your local vault." />

      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        {exchanges.map((ex, i) => {
          const FormComponent = exchangeFormMap[ex.name]
          return (
            <div key={ex.name} className="card" style={{ padding: 0, overflow: 'hidden' }}>
              <div
                style={{
                  display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                  padding: '14px 20px', cursor: 'pointer',
                }}
                onClick={() => setEditing(editing === ex.name ? null : ex.name)}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                  <div style={{
                    width: 8, height: 8, borderRadius: '50%',
                    background: ex.connected ? '#10b981' : 'var(--bg-3)',
                  }} />
                  <div>
                    <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-1)' }}>{ex.name}</div>
                    <div style={{ fontSize: 10, color: 'var(--text-3)' }}>{ex.type}</div>
                  </div>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                  <span style={{
                    fontSize: 10, fontWeight: 600, padding: '3px 8px', borderRadius: 4,
                    color: ex.connected ? '#10b981' : 'var(--text-3)',
                    background: ex.connected ? 'rgba(16,185,129,0.08)' : 'var(--bg-2)',
                  }}>{ex.connected ? 'Connected' : 'Not Connected'}</span>
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--text-3)" strokeWidth="2"
                    style={{ transform: editing === ex.name ? 'rotate(180deg)' : 'rotate(0)', transition: 'transform 0.2s' }}>
                    <polyline points="6 9 12 15 18 9" />
                  </svg>
                </div>
              </div>

              {editing === ex.name && (
                <div style={{ padding: '16px 20px', borderTop: '1px solid var(--border)', display: 'flex', flexDirection: 'column', gap: 12 }}>
                  {FormComponent && <FormComponent fields={ex.fields} onChange={(k, v) => updateField(i, k, v)} />}

                  <div style={{ display: 'flex', gap: 8, marginTop: 4, paddingTop: 12, borderTop: '1px solid var(--border)' }}>
                    <button
                      onClick={() => { const next = [...exchanges]; next[i] = { ...next[i], connected: true }; setExchanges(next) }}
                      style={{
                        padding: '7px 18px', borderRadius: 6, border: 'none', cursor: 'pointer',
                        background: '#6366f1', color: 'white', fontSize: 12, fontWeight: 600,
                      }}
                    >Test & Connect</button>
                    {ex.connected && (
                      <button
                        onClick={() => { const next = [...exchanges]; next[i] = { ...next[i], connected: false }; setExchanges(next) }}
                        style={{
                          padding: '7px 18px', borderRadius: 6, border: '1px solid rgba(239,68,68,0.2)', cursor: 'pointer',
                          background: 'transparent', color: '#ef4444', fontSize: 12, fontWeight: 600,
                        }}
                      >Disconnect</button>
                    )}
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

// Section: Risk Management
function RiskSection() {
  const [maxRisk, setMaxRisk] = useState(2)
  const [maxDailyLoss, setMaxDailyLoss] = useState(5)
  const [maxPositions, setMaxPositions] = useState(5)
  const [maxLeverage, setMaxLeverage] = useState(10)
  const [mode, setMode] = useState('paper')
  const [circuitBreaker, setCircuitBreaker] = useState(true)
  const [maxConsecLosses, setMaxConsecLosses] = useState(3)
  const [cooldown, setCooldown] = useState(30)
  const [drawdownWarn, setDrawdownWarn] = useState(5)
  const [drawdownCrit, setDrawdownCrit] = useState(10)
  const [trailingStop, setTrailingStop] = useState(3)
  const [saved, setSaved] = useState(false)

  return (
    <div>
      <SectionTitle title="Risk Management" desc="Configure position sizing, loss limits, and circuit breaker rules to protect your capital." />

      <div className="card" style={{ padding: '4px 20px', marginBottom: 16 }}>
        <FieldRow label="Trading Mode" desc="Paper mode uses simulated funds. Live mode executes real trades.">
          <SelectInput value={mode} onChange={setMode} options={[
            { value: 'paper', label: 'Paper' },
            { value: 'live', label: 'Live' },
          ]} />
        </FieldRow>
        <FieldRow label="Max Risk Per Trade" desc="Maximum % of portfolio risked on a single trade">
          <NumberInput value={maxRisk} onChange={setMaxRisk} min={0.1} max={10} step={0.1} unit="%" />
        </FieldRow>
        <FieldRow label="Max Daily Loss" desc="Trading halts when daily loss reaches this threshold">
          <NumberInput value={maxDailyLoss} onChange={setMaxDailyLoss} min={1} max={20} step={0.5} unit="%" />
        </FieldRow>
        <FieldRow label="Max Concurrent Positions" desc="Maximum number of open positions at any time">
          <NumberInput value={maxPositions} onChange={setMaxPositions} min={1} max={50} step={1} />
        </FieldRow>
        <FieldRow label="Max Leverage" desc="Maximum leverage allowed across all exchanges">
          <NumberInput value={maxLeverage} onChange={setMaxLeverage} min={1} max={125} step={1} unit="x" />
        </FieldRow>
      </div>

      <h3 style={{ fontSize: 14, fontWeight: 700, color: 'var(--text-1)', marginBottom: 10 }}>Circuit Breaker</h3>
      <div className="card" style={{ padding: '4px 20px', marginBottom: 16 }}>
        <FieldRow label="Enable Circuit Breaker" desc="Auto-halt trading when risk thresholds are breached">
          <Toggle value={circuitBreaker} onChange={setCircuitBreaker} />
        </FieldRow>
        <FieldRow label="Max Consecutive Losses" desc="Pause after N consecutive losing trades">
          <NumberInput value={maxConsecLosses} onChange={setMaxConsecLosses} min={1} max={20} step={1} />
        </FieldRow>
        <FieldRow label="Cooldown Period" desc="Auto-resume after this many minutes">
          <NumberInput value={cooldown} onChange={setCooldown} min={5} max={1440} step={5} unit="min" />
        </FieldRow>
      </div>

      <h3 style={{ fontSize: 14, fontWeight: 700, color: 'var(--text-1)', marginBottom: 10 }}>Drawdown Alerts</h3>
      <div className="card" style={{ padding: '4px 20px', marginBottom: 20 }}>
        <FieldRow label="Warning Threshold" desc="Alert when drawdown from peak reaches this level">
          <NumberInput value={drawdownWarn} onChange={setDrawdownWarn} min={1} max={20} step={0.5} unit="%" />
        </FieldRow>
        <FieldRow label="Critical Threshold" desc="Critical alert & reduce exposure at this level">
          <NumberInput value={drawdownCrit} onChange={setDrawdownCrit} min={2} max={50} step={0.5} unit="%" />
        </FieldRow>
        <FieldRow label="Default Trailing Stop" desc="Trailing stop distance for auto risk management">
          <NumberInput value={trailingStop} onChange={setTrailingStop} min={0.5} max={20} step={0.1} unit="%" />
        </FieldRow>
      </div>

      <SaveButton onClick={() => { setSaved(true); setTimeout(() => setSaved(false), 2000) }} saved={saved} />
    </div>
  )
}

// Section: Agent Config
function AgentSection() {
  const [agentEnabled, setAgentEnabled] = useState(true)
  const [autoTrade, setAutoTrade] = useState(false)
  const [confirmOrders, setConfirmOrders] = useState(true)
  const [minConfidence, setMinConfidence] = useState(70)
  const [scanInterval, setScanInterval] = useState(30)
  const [analystOn, setAnalystOn] = useState(true)
  const [traderOn, setTraderOn] = useState(true)
  const [riskAgentOn, setRiskAgentOn] = useState(true)
  const [researchOn, setResearchOn] = useState(true)
  const [portfolioOn, setPortfolioOn] = useState(true)
  const [saved, setSaved] = useState(false)

  const symbols = [
    { symbol: 'BTC/USDT', exchange: 'Binance', enabled: true },
    { symbol: 'ETH/USDT', exchange: 'Binance', enabled: true },
    { symbol: 'SOL/USDT', exchange: 'Bybit', enabled: true },
    { symbol: 'EUR/USD', exchange: 'MT5', enabled: true },
    { symbol: 'XAU/USD', exchange: 'MT5', enabled: false },
    { symbol: 'AAPL', exchange: 'IBKR', enabled: false },
    { symbol: 'NVDA', exchange: 'IBKR', enabled: true },
  ]
  const [watchlist, setWatchlist] = useState(symbols)

  return (
    <div>
      <SectionTitle title="Agent Configuration" desc="Configure the AI trading agent's behavior, watchlist, and sub-agent modules." />

      <div className="card" style={{ padding: '4px 20px', marginBottom: 16 }}>
        <FieldRow label="Agent Enabled" desc="Master switch for the AI trading agent">
          <Toggle value={agentEnabled} onChange={setAgentEnabled} />
        </FieldRow>
        <FieldRow label="Auto-Trade" desc="Allow agent to execute trades without manual confirmation">
          <Toggle value={autoTrade} onChange={setAutoTrade} />
        </FieldRow>
        <FieldRow label="Order Confirmation" desc="Require confirmation before executing trades">
          <Toggle value={confirmOrders} onChange={setConfirmOrders} />
        </FieldRow>
        <FieldRow label="Min Confidence" desc="Only execute signals above this confidence threshold">
          <NumberInput value={minConfidence} onChange={setMinConfidence} min={10} max={100} step={5} unit="%" />
        </FieldRow>
        <FieldRow label="Scan Interval" desc="How frequently the agent scans for opportunities">
          <NumberInput value={scanInterval} onChange={setScanInterval} min={5} max={300} step={5} unit="sec" />
        </FieldRow>
      </div>

      <h3 style={{ fontSize: 14, fontWeight: 700, color: 'var(--text-1)', marginBottom: 10 }}>Sub-Agents</h3>
      <div className="card" style={{ padding: '4px 20px', marginBottom: 16 }}>
        <FieldRow label="Analyst Agent" desc="Technical & fundamental analysis"><Toggle value={analystOn} onChange={setAnalystOn} /></FieldRow>
        <FieldRow label="Trader Agent" desc="Order execution & management"><Toggle value={traderOn} onChange={setTraderOn} /></FieldRow>
        <FieldRow label="Risk Agent" desc="Position sizing & risk assessment (recommended: always on)"><Toggle value={riskAgentOn} onChange={setRiskAgentOn} /></FieldRow>
        <FieldRow label="Research Agent" desc="News & sentiment analysis"><Toggle value={researchOn} onChange={setResearchOn} /></FieldRow>
        <FieldRow label="Portfolio Agent" desc="Rebalancing & allocation"><Toggle value={portfolioOn} onChange={setPortfolioOn} /></FieldRow>
      </div>

      <h3 style={{ fontSize: 14, fontWeight: 700, color: 'var(--text-1)', marginBottom: 10 }}>Watchlist</h3>
      <div className="card" style={{ padding: 0, overflow: 'hidden', marginBottom: 20 }}>
        <div style={{ padding: '10px 20px', borderBottom: '1px solid var(--border)', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <span style={{ fontSize: 11, color: 'var(--text-3)', fontWeight: 600 }}>
            {watchlist.filter(w => w.enabled).length} of {watchlist.length} active
          </span>
          <button style={{
            padding: '4px 10px', borderRadius: 5, border: '1px solid var(--border)', cursor: 'pointer',
            background: 'transparent', color: '#818cf8', fontSize: 11, fontWeight: 600,
          }}>+ Add Symbol</button>
        </div>
        {watchlist.map((w, i) => (
          <div key={w.symbol} style={{
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
            padding: '10px 20px', borderBottom: '1px solid rgba(255,255,255,0.025)',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-1)' }}>{w.symbol}</span>
              <span style={{ fontSize: 9, color: 'var(--text-3)' }}>{w.exchange}</span>
            </div>
            <Toggle value={w.enabled} onChange={v => {
              const next = [...watchlist]; next[i] = { ...next[i], enabled: v }; setWatchlist(next)
            }} />
          </div>
        ))}
      </div>

      <SaveButton onClick={() => { setSaved(true); setTimeout(() => setSaved(false), 2000) }} saved={saved} />
    </div>
  )
}

// Section: Notifications
function NotificationsSection() {
  const [tradeAlerts, setTradeAlerts] = useState(true)
  const [riskAlerts, setRiskAlerts] = useState(true)
  const [pnlAlerts, setPnlAlerts] = useState(true)
  const [systemAlerts, setSystemAlerts] = useState(true)
  const [telegram, setTelegram] = useState(false)
  const [discord, setDiscord] = useState(false)
  const [email, setEmail] = useState(false)
  const [telegramToken, setTelegramToken] = useState('')
  const [telegramChatId, setTelegramChatId] = useState('')
  const [discordWebhook, setDiscordWebhook] = useState('')
  const [emailAddr, setEmailAddr] = useState('')
  const [saved, setSaved] = useState(false)

  return (
    <div>
      <SectionTitle title="Notifications" desc="Configure which alerts you receive and through which channels." />

      <h3 style={{ fontSize: 14, fontWeight: 700, color: 'var(--text-1)', marginBottom: 10 }}>Alert Types</h3>
      <div className="card" style={{ padding: '4px 20px', marginBottom: 16 }}>
        <FieldRow label="Trade Execution" desc="Notify on trade open, close, and modifications"><Toggle value={tradeAlerts} onChange={setTradeAlerts} /></FieldRow>
        <FieldRow label="Risk Warnings" desc="Drawdown alerts, circuit breaker triggers"><Toggle value={riskAlerts} onChange={setRiskAlerts} /></FieldRow>
        <FieldRow label="P&L Milestones" desc="Daily P&L summary, target hits"><Toggle value={pnlAlerts} onChange={setPnlAlerts} /></FieldRow>
        <FieldRow label="System Status" desc="Exchange disconnections, agent errors"><Toggle value={systemAlerts} onChange={setSystemAlerts} /></FieldRow>
      </div>

      <h3 style={{ fontSize: 14, fontWeight: 700, color: 'var(--text-1)', marginBottom: 10 }}>Channels</h3>
      <div className="card" style={{ padding: '4px 20px', marginBottom: 20 }}>
        <FieldRow label="Telegram" desc="Receive alerts via Telegram bot">
          <Toggle value={telegram} onChange={setTelegram} />
        </FieldRow>
        {telegram && (
          <div style={{ padding: '0 0 14px', display: 'flex', flexDirection: 'column', gap: 8 }}>
            <div>
              <label style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-3)', display: 'block', marginBottom: 4 }}>Bot Token</label>
              <TextInput value={telegramToken} onChange={setTelegramToken} placeholder="123456:ABC-DEF..." />
            </div>
            <div>
              <label style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-3)', display: 'block', marginBottom: 4 }}>Chat ID</label>
              <TextInput value={telegramChatId} onChange={setTelegramChatId} placeholder="-1001234567890" />
            </div>
          </div>
        )}
        <FieldRow label="Discord" desc="Post alerts to a Discord channel">
          <Toggle value={discord} onChange={setDiscord} />
        </FieldRow>
        {discord && (
          <div style={{ padding: '0 0 14px' }}>
            <label style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-3)', display: 'block', marginBottom: 4 }}>Webhook URL</label>
            <TextInput value={discordWebhook} onChange={setDiscordWebhook} placeholder="https://discord.com/api/webhooks/..." />
          </div>
        )}
        <FieldRow label="Email" desc="Daily digest and critical alerts via email">
          <Toggle value={email} onChange={setEmail} />
        </FieldRow>
        {email && (
          <div style={{ padding: '0 0 14px' }}>
            <label style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-3)', display: 'block', marginBottom: 4 }}>Email Address</label>
            <TextInput value={emailAddr} onChange={setEmailAddr} placeholder="trader@example.com" type="email" />
          </div>
        )}
      </div>

      <SaveButton onClick={() => { setSaved(true); setTimeout(() => setSaved(false), 2000) }} saved={saved} />
    </div>
  )
}

// Section: General
function GeneralSection() {
  const [theme, setTheme] = useState('dark')
  const [language, setLanguage] = useState('en')
  const [timezone, setTimezone] = useState('UTC')
  const [saved, setSaved] = useState(false)

  return (
    <div>
      <SectionTitle title="General" desc="Application preferences and display settings." />

      <div className="card" style={{ padding: '4px 20px', marginBottom: 16 }}>
        <FieldRow label="Theme" desc="Dashboard appearance">
          <SelectInput value={theme} onChange={setTheme} options={[
            { value: 'dark', label: 'Dark' },
            { value: 'light', label: 'Light (coming soon)' },
          ]} />
        </FieldRow>
        <FieldRow label="Language" desc="Interface language">
          <SelectInput value={language} onChange={setLanguage} options={[
            { value: 'en', label: 'English' },
            { value: 'vi', label: 'Tiếng Việt' },
            { value: 'zh', label: '中文' },
            { value: 'ja', label: '日本語' },
          ]} />
        </FieldRow>
        <FieldRow label="Timezone" desc="Display times in this timezone">
          <SelectInput value={timezone} onChange={setTimezone} options={[
            { value: 'UTC', label: 'UTC' },
            { value: 'Asia/Ho_Chi_Minh', label: 'Asia/Ho Chi Minh (UTC+7)' },
            { value: 'America/New_York', label: 'US Eastern (UTC-5)' },
            { value: 'Europe/London', label: 'London (UTC+0)' },
            { value: 'Asia/Tokyo', label: 'Tokyo (UTC+9)' },
          ]} />
        </FieldRow>
      </div>

      <h3 style={{ fontSize: 14, fontWeight: 700, color: 'var(--text-1)', marginBottom: 10 }}>About</h3>
      <div className="card" style={{ padding: '16px 20px' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
          <span style={{ fontSize: 12, color: 'var(--text-3)' }}>Version</span>
          <span className="mono" style={{ fontSize: 12, color: 'var(--text-1)' }}>v0.1.0-beta</span>
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
          <span style={{ fontSize: 12, color: 'var(--text-3)' }}>Backend</span>
          <span className="mono" style={{ fontSize: 12, color: 'var(--text-1)' }}>Go 1.23</span>
        </div>
        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
          <span style={{ fontSize: 12, color: 'var(--text-3)' }}>License</span>
          <span style={{ fontSize: 12, color: '#818cf8' }}>Open Source (MIT)</span>
        </div>
      </div>

      <div style={{ marginTop: 20 }}>
        <SaveButton onClick={() => { setSaved(true); setTimeout(() => setSaved(false), 2000) }} saved={saved} />
      </div>
    </div>
  )
}

// Main Settings Panel
export default function SettingsPanel() {
  const [section, setSection] = useState<Section>('exchanges')

  return (
    <div style={{ display: 'flex', gap: 20, height: 'calc(100vh - 108px)', maxWidth: 1100, margin: '0 auto' }}>
      {/* Left nav */}
      <div style={{ width: 220, flexShrink: 0 }}>
        <h1 style={{ fontSize: 20, fontWeight: 800, color: 'var(--text-1)', marginBottom: 20 }}>Settings</h1>
        <nav style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {sections.map(s => {
            const active = section === s.id
            return (
              <button
                key={s.id}
                onClick={() => setSection(s.id)}
                style={{
                  display: 'flex', alignItems: 'center', gap: 10,
                  padding: '10px 12px', borderRadius: 8, border: 'none', cursor: 'pointer',
                  background: active ? 'rgba(99,102,241,0.08)' : 'transparent',
                  color: active ? '#818cf8' : 'var(--text-3)',
                  fontSize: 13, fontWeight: active ? 600 : 500,
                  transition: 'all 0.15s', textAlign: 'left', width: '100%',
                }}
              >
                {s.icon}
                {s.label}
              </button>
            )
          })}
        </nav>
      </div>

      {/* Right content */}
      <div style={{ flex: 1, overflow: 'auto', paddingRight: 8 }}>
        {section === 'exchanges' && <ExchangesSection />}
        {section === 'risk' && <RiskSection />}
        {section === 'agent' && <AgentSection />}
        {section === 'notifications' && <NotificationsSection />}
        {section === 'general' && <GeneralSection />}
      </div>
    </div>
  )
}
