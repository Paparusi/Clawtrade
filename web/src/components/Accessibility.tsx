import { useState, useCallback } from 'react';

// Color vision modes
export type ColorMode = 'normal' | 'protanopia' | 'deuteranopia' | 'tritanopia' | 'high_contrast';

export interface AccessibilitySettings {
  colorMode: ColorMode;
  fontSize: 'small' | 'medium' | 'large';
  reducedMotion: boolean;
  screenReaderMode: boolean;
}

// Color palettes for different vision types
export const COLOR_PALETTES: Record<ColorMode, { positive: string; negative: string; neutral: string; accent: string }> = {
  normal:        { positive: '#22c55e', negative: '#ef4444', neutral: '#94a3b8', accent: '#3b82f6' },
  protanopia:    { positive: '#2563eb', negative: '#f59e0b', neutral: '#94a3b8', accent: '#8b5cf6' },
  deuteranopia:  { positive: '#2563eb', negative: '#f59e0b', neutral: '#94a3b8', accent: '#8b5cf6' },
  tritanopia:    { positive: '#ec4899', negative: '#14b8a6', neutral: '#94a3b8', accent: '#f59e0b' },
  high_contrast: { positive: '#00ff00', negative: '#ff0000', neutral: '#ffffff', accent: '#00ffff' },
};

// Font size mappings
const FONT_SIZES = { small: '14px', medium: '16px', large: '20px' };

// Skip to content link (screen reader)
export function SkipToContent({ targetId }: { targetId: string }) {
  return (
    <a
      href={`#${targetId}`}
      className="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-50 focus:px-4 focus:py-2 focus:bg-blue-600 focus:text-white focus:rounded"
    >
      Skip to content
    </a>
  );
}

// Accessibility settings panel
export function AccessibilityPanel({ settings, onChange }: {
  settings: AccessibilitySettings;
  onChange: (settings: AccessibilitySettings) => void;
}) {
  return (
    <div className="bg-slate-800 rounded-lg border border-slate-700 p-4 space-y-4" role="region" aria-label="Accessibility Settings">
      {/* Color mode selector */}
      <fieldset>
        <legend className="text-sm font-medium text-white mb-2">Color Mode</legend>
        <div className="flex flex-wrap gap-2">
          {(Object.keys(COLOR_PALETTES) as ColorMode[]).map(mode => (
            <button
              key={mode}
              onClick={() => onChange({ ...settings, colorMode: mode })}
              className={`px-3 py-1 rounded text-xs ${settings.colorMode === mode ? 'bg-blue-600 text-white' : 'bg-slate-700 text-slate-300'}`}
              aria-pressed={settings.colorMode === mode}
            >
              {mode.replace('_', ' ')}
            </button>
          ))}
        </div>
      </fieldset>
      {/* Font size */}
      <fieldset>
        <legend className="text-sm font-medium text-white mb-2">Font Size</legend>
        <div className="flex gap-2">
          {(['small', 'medium', 'large'] as const).map(size => (
            <button
              key={size}
              onClick={() => onChange({ ...settings, fontSize: size })}
              className={`px-3 py-1 rounded text-xs ${settings.fontSize === size ? 'bg-blue-600 text-white' : 'bg-slate-700 text-slate-300'}`}
              aria-pressed={settings.fontSize === size}
            >
              {size}
            </button>
          ))}
        </div>
      </fieldset>
      {/* Toggles */}
      <label className="flex items-center gap-2 text-sm text-slate-300 cursor-pointer">
        <input type="checkbox" checked={settings.reducedMotion} onChange={e => onChange({ ...settings, reducedMotion: e.target.checked })} className="rounded" />
        Reduced Motion
      </label>
      <label className="flex items-center gap-2 text-sm text-slate-300 cursor-pointer">
        <input type="checkbox" checked={settings.screenReaderMode} onChange={e => onChange({ ...settings, screenReaderMode: e.target.checked })} className="rounded" />
        Screen Reader Mode
      </label>
    </div>
  );
}

// Hook to manage accessibility settings
export function useAccessibility() {
  const [settings, setSettings] = useState<AccessibilitySettings>({
    colorMode: 'normal',
    fontSize: 'medium',
    reducedMotion: false,
    screenReaderMode: false,
  });

  const applySettings = useCallback((newSettings: AccessibilitySettings) => {
    setSettings(newSettings);
    document.documentElement.style.fontSize = FONT_SIZES[newSettings.fontSize];
    if (newSettings.reducedMotion) {
      document.documentElement.style.setProperty('--animation-duration', '0s');
    } else {
      document.documentElement.style.removeProperty('--animation-duration');
    }
  }, []);

  const getColor = useCallback((type: 'positive' | 'negative' | 'neutral' | 'accent') => {
    return COLOR_PALETTES[settings.colorMode][type];
  }, [settings.colorMode]);

  return { settings, applySettings, getColor };
}

export { FONT_SIZES };
