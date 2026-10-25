import React from 'react'
import { Wifi, Gauge, Zap, CloudRain, Sliders } from 'lucide-react'

export interface ThrottleConfig {
  id: string
  name: string
  speedLabel: string
  label: string
  bandwidthKbps: number // 0 for unlimited
  latencyMs: number
  jitterMs?: number
  isDynamic?: boolean
}

export const THROTTLE_PRESETS: ThrottleConfig[] = [
  { id: 'unlimited', name: 'Direct', speedLabel: 'Unlimited', label: 'Direct / Unlimited', bandwidthKbps: 0, latencyMs: 0, jitterMs: 0 },
  { id: '5g_cband', name: '5G C-Band', speedLabel: '45M', label: '5G Sub-6 C-Band (45 Mbps)', bandwidthKbps: 45000, latencyMs: 20, jitterMs: 5 },
  { id: 'fiber_25m', name: 'Fiber', speedLabel: '25M', label: 'Fiber High-Speed (25 Mbps)', bandwidthKbps: 25000, latencyMs: 15, jitterMs: 2 },
  { id: 'starlink', name: 'Starlink', speedLabel: '15M', label: 'Starlink LEO Satellite (15 Mbps)', bandwidthKbps: 15000, latencyMs: 45, jitterMs: 30 },
  { id: '4g_good', name: '4G Good', speedLabel: '8M', label: '4G LTE Good (8 Mbps)', bandwidthKbps: 8000, latencyMs: 40, jitterMs: 10 },
  { id: '4g_weak', name: '4G Weak', speedLabel: '3M', label: '4G LTE Weak (3 Mbps)', bandwidthKbps: 3000, latencyMs: 80, jitterMs: 25 },
  { id: '3g_fast', name: '3G Fast', speedLabel: '1.5M', label: '3G Fast (1.5 Mbps)', bandwidthKbps: 1500, latencyMs: 150, jitterMs: 40 },
  { id: '3g_slow', name: '3G Slow', speedLabel: '500k', label: '3G Slow (500 Kbps)', bandwidthKbps: 500, latencyMs: 300, jitterMs: 60 },
  { id: 'edge_2g', name: '2G Edge', speedLabel: '150k', label: '2G Edge (150 Kbps)', bandwidthKbps: 150, latencyMs: 600, jitterMs: 120 },
  { id: 'tunnel_drop', name: 'Tunnel Drop', speedLabel: 'Dynamic', label: 'Tunnel Drop (10M -> 400k)', bandwidthKbps: 400, latencyMs: 250, jitterMs: 80, isDynamic: true },
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

  const handleCustomChange = (bw: number, lat: number, jit?: number) => {
    setIsCustom(true)
    const currentJit = jit !== undefined ? jit : (currentConfig.jitterMs || 0)
    onChangeConfig({
      id: 'custom',
      name: 'Custom',
      speedLabel: `${(bw / 1000).toFixed(1)}M`,
      label: `Custom (${(bw / 1000).toFixed(1)} Mbps, ${lat}ms${currentJit > 0 ? ` ±${currentJit}ms` : ''})`,
      bandwidthKbps: bw,
      latencyMs: lat,
      jitterMs: currentJit,
    })
  }

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-2xl p-3.5 sm:p-4 space-y-3.5">
      {/* Header Row */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5">
        <div className="flex items-center space-x-2">
          <Wifi className="w-4 h-4 text-sky-400 shrink-0" />
          <h4 className="text-xs font-semibold text-slate-200 uppercase tracking-wider">
            Network Throttling Simulator
          </h4>
        </div>
        <div className="inline-flex items-center space-x-1.5 text-xs font-mono text-sky-400 font-semibold bg-slate-950 px-2.5 py-1 rounded-lg border border-slate-800 self-start sm:self-auto whitespace-nowrap shrink-0">
          <Gauge className="w-3.5 h-3.5 text-sky-400 shrink-0" />
          <span>
            {currentConfig.bandwidthKbps === 0
              ? 'Unlimited Speed'
              : `${(currentConfig.bandwidthKbps / 1000).toFixed(1)} Mbps (${currentConfig.latencyMs}ms${
                  (currentConfig.jitterMs || 0) > 0 ? ` ±${currentConfig.jitterMs}ms` : ''
                })`}
          </span>
        </div>
      </div>

      {/* Preset Buttons Grid */}
      <div className="grid grid-cols-2 sm:grid-cols-5 gap-2">
        {THROTTLE_PRESETS.map((p) => {
          const isActive = !isCustom && currentConfig.id === p.id
          return (
            <button
              key={p.id}
              onClick={() => handleSelectPreset(p)}
              className={`p-2 sm:p-2.5 rounded-xl text-left text-xs transition-all cursor-pointer flex flex-col justify-between ${
                isActive
                  ? 'bg-sky-500/20 text-sky-300 border border-sky-500/40 shadow-sm shadow-sky-500/10'
                  : 'bg-slate-950 text-slate-400 border border-slate-800/80 hover:bg-slate-800/50 hover:text-slate-200'
              }`}
            >
              <div className="flex items-center justify-between gap-1">
                <span className="font-semibold text-white truncate text-[11px] sm:text-xs">{p.name}</span>
                <span className="text-[10px] font-mono text-sky-400 shrink-0 font-medium">{p.speedLabel}</span>
              </div>
              <div className="text-[10px] text-slate-500 mt-1 truncate">
                {p.bandwidthKbps === 0 ? 'Direct pipe' : `${p.latencyMs}ms RTT${p.jitterMs ? ` ±${p.jitterMs}ms` : ''}`}
              </div>
            </button>
          )
        })}
      </div>

      {/* Custom Slider Cards Grid */}
      <div className="pt-3 border-t border-slate-800/60 space-y-2.5">
        <div className="flex items-center space-x-2 text-xs font-semibold text-slate-300">
          <Sliders className="w-3.5 h-3.5 text-indigo-400 shrink-0" />
          <span>Custom Limits:</span>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-2.5 sm:gap-3">
          {/* Bandwidth Card */}
          <div className="bg-slate-950 p-2.5 rounded-xl border border-slate-800/80 space-y-1.5">
            <div className="flex items-center justify-between text-xs">
              <span className="text-slate-400 text-[11px]">Bandwidth</span>
              <span className="font-mono text-sky-400 font-bold text-xs">
                {currentConfig.bandwidthKbps === 0 ? 'Unlimited' : `${(currentConfig.bandwidthKbps / 1000).toFixed(1)} Mbps`}
              </span>
            </div>
            <input
              type="range"
              min="100"
              max="50000"
              step="100"
              value={currentConfig.bandwidthKbps || 50000}
              onChange={(e) => handleCustomChange(parseInt(e.target.value, 10), currentConfig.latencyMs, currentConfig.jitterMs)}
              className="w-full accent-sky-500 h-1.5 bg-slate-800 rounded-lg cursor-pointer"
            />
          </div>

          {/* Latency Card */}
          <div className="bg-slate-950 p-2.5 rounded-xl border border-slate-800/80 space-y-1.5">
            <div className="flex items-center justify-between text-xs">
              <span className="text-slate-400 text-[11px]">RTT Latency</span>
              <span className="font-mono text-indigo-400 font-bold text-xs">
                {currentConfig.latencyMs} ms
              </span>
            </div>
            <input
              type="range"
              min="0"
              max="800"
              step="10"
              value={currentConfig.latencyMs}
              onChange={(e) => handleCustomChange(currentConfig.bandwidthKbps, parseInt(e.target.value, 10), currentConfig.jitterMs)}
              className="w-full accent-indigo-500 h-1.5 bg-slate-800 rounded-lg cursor-pointer"
            />
          </div>

          {/* Jitter Card */}
          <div className="bg-slate-950 p-2.5 rounded-xl border border-slate-800/80 space-y-1.5">
            <div className="flex items-center justify-between text-xs">
              <span className="text-slate-400 text-[11px]">Jitter Variance</span>
              <span className="font-mono text-amber-400 font-bold text-xs">
                ±{currentConfig.jitterMs || 0} ms
              </span>
            </div>
            <input
              type="range"
              min="0"
              max="150"
              step="5"
              value={currentConfig.jitterMs || 0}
              onChange={(e) => handleCustomChange(currentConfig.bandwidthKbps, currentConfig.latencyMs, parseInt(e.target.value, 10))}
              className="w-full accent-amber-500 h-1.5 bg-slate-800 rounded-lg cursor-pointer"
            />
          </div>
        </div>
      </div>
    </div>
  )
}
