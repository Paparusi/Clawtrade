import { createContext, useContext } from 'react'

// Market state
export type MarketCondition = 'bullish' | 'bearish' | 'sideways' | 'volatile' | 'unknown';
export type EmotionalState = 'calm' | 'fomo' | 'fear' | 'tilt' | 'greed';
export type RiskLevel = 'low' | 'medium' | 'high' | 'critical';

export interface MarketContext {
  condition: MarketCondition;
  volatility: number;       // 0-100
  trendStrength: number;    // 0-100
  sentiment: number;        // -100 to 100
}

export interface UserContext {
  emotionalState: EmotionalState;
  riskLevel: RiskLevel;
  recentPnL: number;
  winStreak: number;
  lossStreak: number;
  sessionDuration: number;  // minutes
}

export interface UIAdaptation {
  theme: 'default' | 'caution' | 'danger' | 'opportunity';
  borderColor: string;
  accentColor: string;
  showWarning: boolean;
  warningMessage?: string;
  tradingEnabled: boolean;
  maxOrderSize?: number;
  suggestions: string[];
}

// Context for UI adaptation
const AdaptationContext = createContext<UIAdaptation | null>(null);

export function useAdaptation(): UIAdaptation {
  const ctx = useContext(AdaptationContext);
  if (!ctx) return getDefaultAdaptation();
  return ctx;
}

// Compute UI adaptation from market + user context
export function computeAdaptation(market: MarketContext, user: UserContext): UIAdaptation {
  const adaptation: UIAdaptation = {
    theme: 'default',
    borderColor: 'border-slate-700',
    accentColor: 'text-blue-400',
    showWarning: false,
    tradingEnabled: true,
    suggestions: [],
  };

  // Emotional state adaptations
  if (user.emotionalState === 'tilt') {
    adaptation.theme = 'danger';
    adaptation.borderColor = 'border-red-500/50';
    adaptation.accentColor = 'text-red-400';
    adaptation.showWarning = true;
    adaptation.warningMessage = 'TILT detected. Consider taking a break before making trades.';
    adaptation.tradingEnabled = false;
    adaptation.suggestions.push('Take a 15-minute break');
    adaptation.suggestions.push('Review your trading journal');
  } else if (user.emotionalState === 'fomo') {
    adaptation.theme = 'caution';
    adaptation.borderColor = 'border-yellow-500/50';
    adaptation.accentColor = 'text-yellow-400';
    adaptation.showWarning = true;
    adaptation.warningMessage = 'FOMO detected. Verify your analysis before entering.';
    adaptation.suggestions.push('Check if this aligns with your strategy');
    adaptation.suggestions.push('Reduce position size by 50%');
  } else if (user.emotionalState === 'fear') {
    adaptation.theme = 'caution';
    adaptation.borderColor = 'border-orange-500/50';
    adaptation.accentColor = 'text-orange-400';
    adaptation.showWarning = true;
    adaptation.warningMessage = 'Fear detected. Review positions calmly.';
    adaptation.suggestions.push('Check your stop losses are set');
  } else if (user.emotionalState === 'greed') {
    adaptation.theme = 'caution';
    adaptation.borderColor = 'border-yellow-500/50';
    adaptation.accentColor = 'text-yellow-400';
    adaptation.showWarning = true;
    adaptation.warningMessage = 'Greed detected. Stick to your risk limits.';
    adaptation.suggestions.push('Take partial profits');
    adaptation.suggestions.push('Do not increase position size');
  }

  // Market condition adaptations
  if (market.condition === 'volatile' && market.volatility > 80) {
    adaptation.suggestions.push('High volatility: widen stop losses or reduce size');
    if (adaptation.theme === 'default') {
      adaptation.theme = 'caution';
      adaptation.borderColor = 'border-yellow-500/30';
    }
  }

  // Risk level
  if (user.riskLevel === 'critical') {
    adaptation.tradingEnabled = false;
    adaptation.showWarning = true;
    adaptation.warningMessage = 'Risk level critical. Trading disabled.';
  } else if (user.riskLevel === 'high') {
    adaptation.suggestions.push('Reduce overall exposure');
  }

  // Session fatigue
  if (user.sessionDuration > 240) {
    adaptation.suggestions.push('Long session detected. Consider taking a break.');
  }

  // Loss streak
  if (user.lossStreak >= 3) {
    adaptation.suggestions.push(`${user.lossStreak} consecutive losses. Review your strategy.`);
    if (!adaptation.showWarning) {
      adaptation.showWarning = true;
      adaptation.warningMessage = 'Losing streak detected. Trade carefully.';
    }
  }

  return adaptation;
}

function getDefaultAdaptation(): UIAdaptation {
  return {
    theme: 'default',
    borderColor: 'border-slate-700',
    accentColor: 'text-blue-400',
    showWarning: false,
    tradingEnabled: true,
    suggestions: [],
  };
}

// Warning banner component
export function WarningBanner({ adaptation }: { adaptation: UIAdaptation }) {
  if (!adaptation.showWarning || !adaptation.warningMessage) return null;

  const bgColor = adaptation.theme === 'danger' ? 'bg-red-900/30 border-red-500/50' : 'bg-yellow-900/30 border-yellow-500/50';
  const textColor = adaptation.theme === 'danger' ? 'text-red-300' : 'text-yellow-300';

  return (
    <div className={`px-4 py-2 ${bgColor} border-b ${textColor} text-sm flex items-center justify-between`}>
      <span>{adaptation.warningMessage}</span>
      {adaptation.suggestions.length > 0 && (
        <span className="text-xs opacity-75">Tip: {adaptation.suggestions[0]}</span>
      )}
    </div>
  );
}

// Suggestions panel component
export function SuggestionsPanel({ suggestions }: { suggestions: string[] }) {
  if (suggestions.length === 0) return null;

  return (
    <div className="bg-slate-800 rounded-lg border border-slate-700 p-3">
      <div className="text-xs text-slate-400 mb-2 font-medium">AI Suggestions</div>
      <ul className="space-y-1">
        {suggestions.map((s, i) => (
          <li key={i} className="text-sm text-slate-300 flex items-start gap-2">
            <span className="text-blue-400 mt-0.5">•</span>
            {s}
          </li>
        ))}
      </ul>
    </div>
  );
}

// Provider component
export function AdaptationProvider({ market, user, children }: {
  market: MarketContext;
  user: UserContext;
  children: React.ReactNode;
}) {
  const adaptation = computeAdaptation(market, user);
  return (
    <AdaptationContext.Provider value={adaptation}>
      {children}
    </AdaptationContext.Provider>
  );
}

export { AdaptationContext };
