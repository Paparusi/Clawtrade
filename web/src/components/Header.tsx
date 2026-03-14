import { useEffect, useState } from 'react'
import { fetchHealth } from '../api/client'

export default function Header() {
  const [status, setStatus] = useState<'connecting' | 'ok' | 'offline'>('connecting')

  useEffect(() => {
    fetchHealth()
      .then((data) => setStatus(data.status === 'ok' ? 'ok' : 'offline'))
      .catch(() => setStatus('offline'))
  }, [])

  return (
    <header style={{
      display: 'flex', alignItems: 'center', justifyContent: 'space-between',
      padding: '0 24px', height: 48,
      borderBottom: '1px solid var(--border)',
      background: 'var(--bg-1)',
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <span style={{ fontSize: 14, fontWeight: 700, color: 'var(--text-1)', letterSpacing: '-0.02em' }}>
          Clawtrade
        </span>
        <span style={{
          fontSize: 10, fontWeight: 600, color: 'var(--accent)', letterSpacing: '0.04em',
          padding: '2px 8px', borderRadius: 6,
          background: 'rgba(99,102,241,0.1)', border: '1px solid rgba(99,102,241,0.15)',
        }}>
          BETA
        </span>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
        {/* Search */}
        <div style={{
          display: 'flex', alignItems: 'center', gap: 8,
          padding: '6px 12px', borderRadius: 8,
          background: 'var(--bg-2)', border: '1px solid var(--border)',
          color: 'var(--text-3)', fontSize: 12, cursor: 'pointer',
        }}>
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>
          <span>Search...</span>
          <kbd style={{ fontSize: 10, padding: '1px 5px', borderRadius: 4, background: 'var(--bg-0)', color: 'var(--text-3)', fontFamily: 'JetBrains Mono' }}>⌘K</kbd>
        </div>

        {/* Status */}
        <div style={{
          display: 'flex', alignItems: 'center', gap: 6,
          padding: '4px 10px', borderRadius: 6,
          fontSize: 11, fontWeight: 500,
          color: status === 'ok' ? '#10b981' : status === 'connecting' ? '#f59e0b' : '#ef4444',
          background: status === 'ok' ? 'rgba(16,185,129,0.08)' : status === 'connecting' ? 'rgba(245,158,11,0.08)' : 'rgba(239,68,68,0.08)',
        }}>
          <div style={{
            width: 6, height: 6, borderRadius: '50%',
            background: status === 'ok' ? '#10b981' : status === 'connecting' ? '#f59e0b' : '#ef4444',
          }} className={status === 'ok' ? 'pulse-dot' : ''} />
          {status === 'ok' ? 'Connected' : status === 'connecting' ? 'Connecting' : 'Offline'}
        </div>
      </div>
    </header>
  )
}
