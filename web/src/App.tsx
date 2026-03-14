import { useState } from 'react'
import Header from './components/Header'
import Sidebar from './components/Sidebar'
import ChatPanel from './components/ChatPanel'
import PositionsTable from './components/PositionsTable'
import PortfolioSummary from './components/PortfolioSummary'
import PriceChart from './components/PriceChart'
import MarketOverview from './components/MarketOverview'
import ExchangeStatus from './components/ExchangeStatus'
import AgentStatus from './components/AgentStatus'
import SettingsPanel from './components/SettingsPanel'
import PerformanceChart from './components/PerformanceChart'
import RiskAnalysis from './components/RiskAnalysis'
import AssetAllocation from './components/AssetAllocation'
import MonthlyReturns from './components/MonthlyReturns'
import TopMovers from './components/TopMovers'
import DrawdownChart from './components/DrawdownChart'

export default function App() {
  const [tab, setTab] = useState('dashboard')

  return (
    <div style={{ display: 'flex', height: '100vh', width: '100vw', background: 'var(--bg-0)', color: 'var(--text-2)', overflow: 'hidden' }}>
      <Sidebar activeTab={tab} onTabChange={setTab} />
      <div style={{ display: 'flex', flexDirection: 'column', flex: 1, minWidth: 0 }}>
        <Header />
        <main style={{ flex: 1, overflow: 'auto', padding: 20 }}>
          {tab === 'dashboard' && <DashboardView />}
          {tab === 'chat' && (
            <div style={{ height: 'calc(100vh - 68px)', maxWidth: 900, margin: '0 auto' }}>
              <ChatPanel />
            </div>
          )}
          {tab === 'positions' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              <PortfolioSummary />
              <PositionsTable />
            </div>
          )}
          {tab === 'analytics' && <AnalyticsView />}
          {tab === 'strategies' && <StrategiesPlaceholder />}
          {tab === 'settings' && <SettingsPanel />}
        </main>
      </div>
    </div>
  )
}

function DashboardView() {
  return (
    <div style={{
      display: 'grid',
      gridTemplateColumns: '1fr 380px',
      gridTemplateRows: 'auto minmax(0, 2fr) minmax(0, 3fr)',
      gap: 12,
      width: '100%',
      height: 'calc(100vh - 88px)',
      minHeight: 0,
    }}>
      {/* Row 1: Portfolio Stats - full width */}
      <div style={{ gridColumn: '1 / -1' }}>
        <PortfolioSummary />
      </div>

      {/* Row 2 Left: Chart */}
      <PriceChart />

      {/* Row 2 Right: Agent */}
      <AgentStatus />

      {/* Row 3 Left: Positions */}
      <PositionsTable />

      {/* Row 3 Right: Exchanges + Markets */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12, minHeight: 0, overflow: 'hidden' }}>
        <ExchangeStatus />
        <div style={{ flex: 1, minHeight: 0 }}>
          <MarketOverview />
        </div>
      </div>
    </div>
  )
}

function AnalyticsView() {
  return (
    <div style={{
      display: 'grid',
      gridTemplateColumns: '1fr 1fr',
      gridTemplateRows: 'minmax(280px, 1fr) minmax(220px, auto) minmax(180px, auto)',
      gap: 12,
      width: '100%',
      height: 'calc(100vh - 88px)',
      minHeight: 0,
    }}>
      {/* Row 1: Performance Chart + Risk Analysis */}
      <PerformanceChart />
      <RiskAnalysis />

      {/* Row 2: Asset Allocation + Monthly Returns + Top Movers */}
      <div style={{ gridColumn: '1 / -1', display: 'grid', gridTemplateColumns: '1fr 1.2fr 1fr', gap: 12, minHeight: 0 }}>
        <AssetAllocation />
        <MonthlyReturns />
        <TopMovers />
      </div>

      {/* Row 3: Drawdown Chart - full width */}
      <div style={{ gridColumn: '1 / -1', minHeight: 0 }}>
        <DrawdownChart />
      </div>
    </div>
  )
}

function StrategiesPlaceholder() {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: 'calc(100vh - 120px)' }}>
      <div className="card" style={{ textAlign: 'center', padding: 48, maxWidth: 440 }}>
        <div style={{
          width: 56, height: 56, borderRadius: 14, margin: '0 auto 20px',
          background: 'rgba(99,102,241,0.08)', border: '1px solid rgba(99,102,241,0.15)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}>
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#818cf8" strokeWidth="2" strokeLinecap="round">
            <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/>
          </svg>
        </div>
        <h2 style={{ fontSize: 18, fontWeight: 700, color: 'var(--text-1)', marginBottom: 8 }}>Strategy Arena</h2>
        <p style={{ fontSize: 13, color: 'var(--text-3)', lineHeight: 1.6 }}>
          Create, backtest, and A/B test trading strategies across Crypto, Forex, Stocks and Indices.
        </p>
      </div>
    </div>
  )
}
