import { useState } from 'react'
import Header from './components/Header'
import Sidebar from './components/Sidebar'
import ChatPanel from './components/ChatPanel'
import PositionsTable from './components/PositionsTable'
import PortfolioSummary from './components/PortfolioSummary'
import TradingViewChart from './components/TradingViewChart'
import MarketOverview from './components/MarketOverview'
// @ts-expect-error reserved for future use
import ExchangeStatus from './components/ExchangeStatus'
// @ts-expect-error reserved for future use
import AgentStatus from './components/AgentStatus'
import AgentInsights from './components/AgentInsights'
import RecentTrades from './components/RecentTrades'
import TradeToast from './components/TradeToast'
import StatusBar from './components/StatusBar'
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
    <div className="flex h-screen w-screen bg-[var(--bg-0)] text-[var(--text-2)] overflow-hidden">
      <TradeToast />
      <Sidebar activeTab={tab} onTabChange={setTab} />
      <div className="flex flex-col flex-1 min-w-0">
        <Header />
        <main className="flex-1 overflow-auto p-3">
          {tab === 'dashboard' && <DashboardView />}
          {tab === 'chat' && (
            <div className="h-[calc(100vh-68px)] max-w-[900px] mx-auto">
              <ChatPanel />
            </div>
          )}
          {tab === 'positions' && (
            <div className="flex flex-col gap-3">
              <PortfolioSummary />
              <PositionsTable />
            </div>
          )}
          {tab === 'analytics' && <AnalyticsView />}
          {tab === 'strategies' && <StrategiesPlaceholder />}
          {tab === 'settings' && <SettingsPanel />}
        </main>
        <StatusBar />
      </div>
    </div>
  )
}

function DashboardView() {
  return (
    <div className="flex gap-3 h-[calc(100vh-100px)]">
      {/* Main area */}
      <div className="flex-1 flex flex-col gap-3 min-w-0">
        {/* Portfolio stats */}
        <PortfolioSummary />

        {/* TradingView chart */}
        <div className="flex-1 min-h-0">
          <TradingViewChart />
        </div>

        {/* Bottom row: Market + Trades */}
        <div className="grid grid-cols-2 gap-3" style={{ height: '220px' }}>
          <MarketOverview />
          <RecentTrades />
        </div>
      </div>

      {/* Right sidebar */}
      <div className="w-[340px] flex flex-col gap-3 shrink-0">
        {/* Chat */}
        <div className="flex-1 min-h-0 card overflow-hidden">
          <ChatPanel />
        </div>

        {/* Agent Insights */}
        <div className="h-[280px] card overflow-hidden">
          <AgentInsights />
        </div>
      </div>
    </div>
  )
}

function AnalyticsView() {
  return (
    <div
      className="grid gap-3 w-full min-h-0"
      style={{
        gridTemplateColumns: '1fr 1fr',
        gridTemplateRows: 'minmax(280px, 1fr) minmax(220px, auto) minmax(180px, auto)',
        height: 'calc(100vh - 100px)',
      }}
    >
      <PerformanceChart />
      <RiskAnalysis />
      <div className="col-span-2 grid gap-3 min-h-0" style={{ gridTemplateColumns: '1fr 1.2fr 1fr' }}>
        <AssetAllocation />
        <MonthlyReturns />
        <TopMovers />
      </div>
      <div className="col-span-2 min-h-0">
        <DrawdownChart />
      </div>
    </div>
  )
}

function StrategiesPlaceholder() {
  return (
    <div className="flex items-center justify-center" style={{ height: 'calc(100vh - 120px)' }}>
      <div className="card text-center p-12 max-w-[440px]">
        <div className="w-14 h-14 rounded-[14px] mx-auto mb-5 bg-[rgba(99,102,241,0.08)] border border-[rgba(99,102,241,0.15)] flex items-center justify-center">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#818cf8" strokeWidth="2" strokeLinecap="round">
            <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/>
          </svg>
        </div>
        <h2 className="text-lg font-bold text-[var(--text-1)] mb-2">Strategy Arena</h2>
        <p className="text-[13px] text-[var(--text-3)] leading-relaxed">
          Create, backtest, and A/B test trading strategies across Crypto, Forex, Stocks and Indices.
        </p>
      </div>
    </div>
  )
}
