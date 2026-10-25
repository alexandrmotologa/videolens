import React from 'react'
import { Wifi, Gauge, Zap, CloudRain, Sliders } from 'lucide-react'

export interface ThrottleConfig {
  id: string
  label: string
  bandwidthKbps: number // 0 for unlimited
  latencyMs: number
  isDynamic?: boolean
}

export const THROTTLE_PRESETS: ThrottleConfig[] = [
  { id: 'unlimited', label: 'Direct / Unlimited', bandwidthKbps: 0, latencyMs: 0 },
  { id: 'fiber_25m', label: 'Fiber High-Speed (25 Mbps)', bandwidthKbps: 25000, latencyMs: 15 },
  { id: '4g_good', label: '4G LTE Good (8 Mbps)', bandwidthKbps: 8000, latencyMs: 40 },
  { id: '4g_weak', label: '4G LTE Weak (3 Mbps)', bandwidthKbps: 3000, latencyMs: 80 },
  { id: '3g_fast', label: '3G Fast (1.5 Mbps)', bandwidthKbps: 1500, latencyMs: 150 },
  { id: '3g_slow', label: '3G Slow (500 Kbps)', bandwidthKbps: 500, latencyMs: 300 },
  { id: 'edge_2g', label: '2G Edge (150 Kbps)', bandwidthKbps: 150, latencyMs: 600 },
  { id: 'tunnel_drop', label: 'Tunnel Drop (10M -> 400k)', bandwidthKbps: 400, latencyMs: 250, isDynamic: true },
]

interface ThrottleControlProps {
  currentConfig: ThrottleConfig
  onChangeConfig: (cfg: ThrottleConfig) => void
}

export const ThrottleControl: React.FC<ThrottleControlProps> = ({ currentConfig, onChangeConfig }) => {
  const [isCustom, setIsCustom] = React.useState(false)

  const handleSelectPreset = (preset: ThrottleConfig) => {
    setIsCustom(false)
    onChangeConfig(preset)
  }

  const handleCustomChange = (bw: number, lat: number) => {
    setIsCustom(true)
    onChangeConfig({
      id: 'custom',
      label: `Custom (${(bw / 1000).toFixed(1)} Mbps, ${lat}ms)`,
      bandwidthKbps: bw,
      latencyMs: lat,
    })
  }

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-2xl p-4 space-y-3.5">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <Wifi className="w-4 h-4 text-sky-400" />
          <h4 className="text-xs font-semibold text-slate-200 uppercase tracking-wider">
            Network Bandwidth Throttling Simulator
          </h4>
        </div>
        <div className="flex items-center space-x-1.5 text-xs font-mono text-sky-400 font-semibold bg-slate-950 px-2.5 py-1 rounded-lg border border-slate-800">
          <Gauge className="w-3.5 h-3.5 text-sky-400" />
          <span>
            {currentConfig.bandwidthKbps === 0
              ? 'Unlimited Speed'
              : `${(currentConfig.bandwidthKbps / 1000).toFixed(1)} Mbps (${currentConfig.latencyMs}ms RTT)`}
          </span>
        </div>
      </div>

      {/* Preset Buttons Grid */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
        {THROTTLE_PRESETS.map((p) => {
          const isActive = !isCustom && currentConfig.id === p.id
          return (
            <button
              key={p.id}
              onClick={() => handleSelectPreset(p)}
              className={`px-3 py-2 rounded-xl text-left text-xs font-medium transition-all ${
                isActive
                  ? 'bg-sky-500/20 text-sky-300 border border-sky-500/40 shadow-sm shadow-sky-500/10'
                  : 'bg-slate-950 text-slate-400 border border-slate-800/80 hover:bg-slate-800/50 hover:text-slate-200'
              }`}
            >
              <div className="font-semibold truncate">{p.label}</div>
              <div className="text-[10px] text-slate-500 mt-0.5">
                {p.bandwidthKbps === 0 ? 'No artificial throttling' : `${p.latencyMs}ms injected latency`}
              </div>
            </button>
          )
        })}
      </div>

      {/* Custom Slider Toggle */}
      <div className="pt-2 border-t border-slate-800/60 flex flex-col sm:flex-row items-center gap-4 text-xs">
        <div className="flex items-center space-x-2 w-full sm:w-auto">
          <Sliders className="w-3.5 h-3.5 text-indigo-400" />
          <span className="text-slate-400">Custom Limit:</span>
        </div>

        <div className="flex-1 flex items-center space-x-3 w-full">
          <span className="text-[11px] text-slate-500 w-16">Bandwidth:</span>
          <input
            type="range"
            min="100"
            max="30000"
            step="100"
            value={currentConfig.bandwidthKbps || 30000}
            onChange={(e) => handleCustomChange(parseInt(e.target.value, 10), currentConfig.latencyMs)}
            className="flex-1 accent-sky-500"
          />
          <span className="text-[11px] font-mono text-sky-300 w-20 text-right">
            {(currentConfig.bandwidthKbps / 1000).toFixed(1)} Mbps
          </span>
        </div>

        <div className="flex-1 flex items-center space-x-3 w-full">
          <span className="text-[11px] text-slate-500 w-16">Latency:</span>
          <input
            type="range"
            min="0"
            max="800"
            step="10"
            value={currentConfig.latencyMs}
            onChange={(e) => handleCustomChange(currentConfig.bandwidthKbps, parseInt(e.target.value, 10))}
            className="flex-1 accent-indigo-500"
          />
          <span className="text-[11px] font-mono text-indigo-300 w-16 text-right">
            {currentConfig.latencyMs} ms
          </span>
        </div>
      </div>
    </div>
  )
}
