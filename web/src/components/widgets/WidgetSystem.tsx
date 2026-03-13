import { useState, useCallback } from 'react'

// Widget definition
export interface WidgetConfig {
  id: string;
  type: string;           // e.g., "chart", "positions", "chat", "portfolio", "news"
  title: string;
  x: number;              // grid column
  y: number;              // grid row
  width: number;          // grid columns span
  height: number;         // grid rows span
  minimized?: boolean;
  settings?: Record<string, unknown>;
}

// Layout definition
export interface LayoutConfig {
  name: string;
  columns: number;        // grid columns
  rowHeight: number;      // row height in px
  widgets: WidgetConfig[];
}

// Widget registry entry
interface WidgetType {
  type: string;
  name: string;
  icon: string;
  defaultSize: { width: number; height: number };
  component: React.ComponentType<WidgetProps>;
}

export interface WidgetProps {
  config: WidgetConfig;
  onClose?: () => void;
  onMinimize?: () => void;
  onSettings?: () => void;
}

// Widget wrapper with drag handle and controls
function WidgetWrapper({ config, children, onClose, onMinimize }: {
  config: WidgetConfig;
  children: React.ReactNode;
  onClose?: () => void;
  onMinimize?: () => void;
}) {
  return (
    <div
      className="bg-slate-800 rounded-lg border border-slate-700 flex flex-col overflow-hidden"
      style={{
        gridColumn: `${config.x + 1} / span ${config.width}`,
        gridRow: `${config.y + 1} / span ${config.height}`,
      }}
    >
      {/* Header with drag handle and controls */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-slate-700 cursor-move">
        <span className="text-sm font-medium text-white">{config.title}</span>
        <div className="flex gap-1">
          {onMinimize && (
            <button onClick={onMinimize} className="text-slate-400 hover:text-white text-xs px-1">—</button>
          )}
          {onClose && (
            <button onClick={onClose} className="text-slate-400 hover:text-red-400 text-xs px-1">×</button>
          )}
        </div>
      </div>
      {/* Content */}
      {!config.minimized && (
        <div className="flex-1 overflow-auto">{children}</div>
      )}
    </div>
  );
}

// Placeholder widget components
function ChartWidget({ config }: WidgetProps) {
  return <div className="p-4 text-slate-400 text-sm">Chart for {(config.settings?.symbol as string) || 'BTC-USDT'}</div>;
}

function NewsWidget(_props: WidgetProps) {
  return <div className="p-4 text-slate-400 text-sm">News feed</div>;
}

function OrderBookWidget(_props: WidgetProps) {
  return <div className="p-4 text-slate-400 text-sm">Order book</div>;
}

// Simple placeholder components for positions, portfolio, chat
function PositionsWidget(_props: WidgetProps) { return <div className="p-4 text-slate-400 text-sm">Open positions</div>; }
function PortfolioWidget(_props: WidgetProps) { return <div className="p-4 text-slate-400 text-sm">Portfolio summary</div>; }
function ChatWidget(_props: WidgetProps) { return <div className="p-4 text-slate-400 text-sm">AI Chat</div>; }

// Widget type registry
const WIDGET_TYPES: WidgetType[] = [
  { type: 'chart', name: 'Price Chart', icon: '📈', defaultSize: { width: 2, height: 2 }, component: ChartWidget },
  { type: 'positions', name: 'Positions', icon: '📋', defaultSize: { width: 2, height: 1 }, component: PositionsWidget },
  { type: 'portfolio', name: 'Portfolio', icon: '💰', defaultSize: { width: 1, height: 1 }, component: PortfolioWidget },
  { type: 'chat', name: 'AI Chat', icon: '💬', defaultSize: { width: 1, height: 2 }, component: ChatWidget },
  { type: 'news', name: 'News', icon: '📰', defaultSize: { width: 1, height: 1 }, component: NewsWidget },
  { type: 'orderbook', name: 'Order Book', icon: '📊', defaultSize: { width: 1, height: 1 }, component: OrderBookWidget },
];

// Default layouts
const DEFAULT_LAYOUTS: LayoutConfig[] = [
  {
    name: 'Trading',
    columns: 4,
    rowHeight: 200,
    widgets: [
      { id: 'chart-1', type: 'chart', title: 'BTC-USDT', x: 0, y: 0, width: 3, height: 2 },
      { id: 'portfolio-1', type: 'portfolio', title: 'Portfolio', x: 3, y: 0, width: 1, height: 1 },
      { id: 'positions-1', type: 'positions', title: 'Positions', x: 0, y: 2, width: 2, height: 1 },
      { id: 'chat-1', type: 'chat', title: 'AI Chat', x: 2, y: 2, width: 2, height: 1 },
    ],
  },
  {
    name: 'Analysis',
    columns: 3,
    rowHeight: 250,
    widgets: [
      { id: 'chart-1', type: 'chart', title: 'BTC-USDT', x: 0, y: 0, width: 2, height: 2 },
      { id: 'news-1', type: 'news', title: 'News', x: 2, y: 0, width: 1, height: 1 },
      { id: 'chat-1', type: 'chat', title: 'AI Chat', x: 2, y: 1, width: 1, height: 1 },
    ],
  },
];

// Main WidgetGrid component
export default function WidgetGrid() {
  const [layout, setLayout] = useState<LayoutConfig>(DEFAULT_LAYOUTS[0]);
  const [savedLayouts, setSavedLayouts] = useState<LayoutConfig[]>(DEFAULT_LAYOUTS);

  const addWidget = useCallback((type: string) => {
    const widgetType = WIDGET_TYPES.find(w => w.type === type);
    if (!widgetType) return;
    const newWidget: WidgetConfig = {
      id: `${type}-${Date.now()}`,
      type,
      title: widgetType.name,
      x: 0, y: 0,
      width: widgetType.defaultSize.width,
      height: widgetType.defaultSize.height,
    };
    setLayout(prev => ({ ...prev, widgets: [...prev.widgets, newWidget] }));
  }, []);

  const removeWidget = useCallback((id: string) => {
    setLayout(prev => ({ ...prev, widgets: prev.widgets.filter(w => w.id !== id) }));
  }, []);

  const toggleMinimize = useCallback((id: string) => {
    setLayout(prev => ({
      ...prev,
      widgets: prev.widgets.map(w => w.id === id ? { ...w, minimized: !w.minimized } : w),
    }));
  }, []);

  const saveLayout = useCallback((name: string) => {
    const saved = { ...layout, name };
    setSavedLayouts(prev => [...prev.filter(l => l.name !== name), saved]);
  }, [layout]);

  const loadLayout = useCallback((name: string) => {
    const found = savedLayouts.find(l => l.name === name);
    if (found) setLayout(found);
  }, [savedLayouts]);

  const getWidgetComponent = (type: string) => {
    return WIDGET_TYPES.find(w => w.type === type)?.component;
  };

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center gap-2 px-4 py-2 bg-slate-800 border-b border-slate-700">
        <span className="text-sm text-slate-400">Layout:</span>
        {savedLayouts.map(l => (
          <button
            key={l.name}
            onClick={() => loadLayout(l.name)}
            className={`px-3 py-1 rounded text-xs ${layout.name === l.name ? 'bg-blue-600 text-white' : 'bg-slate-700 text-slate-300 hover:bg-slate-600'}`}
          >
            {l.name}
          </button>
        ))}
        <button
          onClick={() => saveLayout(layout.name)}
          className="px-3 py-1 rounded text-xs bg-slate-700 text-slate-300 hover:bg-slate-600"
          title="Save current layout"
        >
          Save
        </button>
        <div className="flex-1" />
        <span className="text-sm text-slate-400">Add:</span>
        {WIDGET_TYPES.map(w => (
          <button
            key={w.type}
            onClick={() => addWidget(w.type)}
            className="px-2 py-1 bg-slate-700 text-slate-300 rounded text-xs hover:bg-slate-600"
            title={w.name}
          >
            {w.icon}
          </button>
        ))}
      </div>
      {/* Grid */}
      <div
        className="flex-1 p-4 gap-4"
        style={{
          display: 'grid',
          gridTemplateColumns: `repeat(${layout.columns}, 1fr)`,
          gridAutoRows: `${layout.rowHeight}px`,
        }}
      >
        {layout.widgets.map(widget => {
          const Component = getWidgetComponent(widget.type);
          return (
            <WidgetWrapper
              key={widget.id}
              config={widget}
              onClose={() => removeWidget(widget.id)}
              onMinimize={() => toggleMinimize(widget.id)}
            >
              {Component && <Component config={widget} />}
            </WidgetWrapper>
          );
        })}
      </div>
    </div>
  );
}

export { WIDGET_TYPES, DEFAULT_LAYOUTS, WidgetWrapper };
