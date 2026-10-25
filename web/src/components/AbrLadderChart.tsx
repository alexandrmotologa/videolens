import React, { useState } from 'react'
import { LadderAnalysis, LadderRung } from '../types/media'
import { BarChart3, AlertTriangle, CheckCircle2, TrendingUp, Sparkles, Plus, Trash2, RotateCcw } from 'lucide-react'

interface AbrLadderChartProps {
  initialAnalysis?: LadderAnalysis
  onReanalyze?: (rungs: LadderRung[]) => Promise<LadderAnalysis | void>
}

export const AbrLadderChart: React.FC<AbrLadderChartProps> = ({ initialAnalysis, onReanalyze }) => {
  const defaultRungs: LadderRung[] = [
    { id: '1080p', resolution: '1920x1080', width: 1920, height: 1080, fps: 30, bitrate: 5500000, bitrateKbps: 5500, bitrateMbps: 5.5, codec: 'avc1.640028', codecFamily: 'H264', bitsPerPixel: 0.0885, stepRatio: 1.83, bppStatus: 'OPTIMAL', evaluation: 'Optimal BPP for 1080p' },
    { id: '720p', resolution: '1280x720', width: 1280, height: 720, fps: 30, bitrate: 3000000, bitrateKbps: 3000, bitrateMbps: 3.0, codec: 'avc1.4d401f', codecFamily: 'H264', bitsPerPixel: 0.1085, stepRatio: 1.87, bppStatus: 'OPTIMAL', evaluation: 'Optimal BPP for 720p' },
    { id: '480p', resolution: '854x480', width: 854, height: 480, fps: 30, bitrate: 1600000, bitrateKbps: 1600, bitrateMbps: 1.6, codec: 'avc1.4d401f', codecFamily: 'H264', bitsPerPixel: 0.1301, stepRatio: 2.0, bppStatus: 'OPTIMAL', evaluation: 'Optimal BPP for 480p' },
    { id: '360p', resolution: '640x360', width: 640, height: 360, fps: 30, bitrate: 800000, bitrateKbps: 800, bitrateMbps: 0.8, codec: 'avc1.4d401e', codecFamily: 'H264', bitsPerPixel: 0.1157, stepRatio: 1.0, bppStatus: 'OPTIMAL', evaluation: 'Optimal BPP for 360p' },
  ]

  const [analysis, setAnalysis] = useState<LadderAnalysis>(
    initialAnalysis || {
      rungs: defaultRungs,
      rungCount: defaultRungs.length,
      minBitrate: 800000,
      maxBitrate: 5500000,
      dynamicRange: 6.875,
      healthScore: 100,
      issues: [],
      recommendations: ['Bitrate ladder follows recommended resolution steps, BPP targets, and scaling intervals.'],
    }
  )

  const [customRungs, setCustomRungs] = useState<LadderRung[]>(analysis.rungs)

  // Update when parent props change
  React.useEffect(() => {
    if (initialAnalysis) {
      setAnalysis(initialAnalysis)
      setCustomRungs(initialAnalysis.rungs)
    }
  }, [initialAnalysis])

  const maxBitrate = Math.max(...customRungs.map((r) => r.bitrate), 1)

  const handleUpdateRung = async (idx: number, field: keyof LadderRung, val: any) => {
    const updated = [...customRungs]
    updated[idx] = { ...updated[idx], [field]: val }

    // Recompute width & height if resolution changes
    if (field === 'resolution' && typeof val === 'string') {
      const parts = val.split('x')
      if (parts.length === 2) {
        updated[idx].width = parseInt(parts[0], 10) || 0
        updated[idx].height = parseInt(parts[1], 10) || 0
      }
    }

    setCustomRungs(updated)
    if (onReanalyze) {
      const res = await onReanalyze(updated)
      if (res) setAnalysis(res)
    }
  }

  const handleAddRung = () => {
    const newRung: LadderRung = {
      id: `tier-${customRungs.length + 1}`,
      resolution: '1280x720',
      width: 1280,
      height: 720,
      fps: 30,
      bitrate: 2500000,
      bitrateKbps: 2500,
      bitrateMbps: 2.5,
      codec: 'avc1.4d401f',
      codecFamily: 'H264',
      bitsPerPixel: 0.09,
      stepRatio: 1.5,
      bppStatus: 'OPTIMAL',
      evaluation: 'Custom tier',
    }
    const updated = [...customRungs, newRung]
    setCustomRungs(updated)
    if (onReanalyze) {
      onReanalyze(updated).then((res) => {
        if (res) setAnalysis(res)
      })
    }
  }

  const handleDeleteRung = (idx: number) => {
    const updated = customRungs.filter((_, i) => i !== idx)
    setCustomRungs(updated)
    if (onReanalyze) {
      onReanalyze(updated).then((res) => {
        if (res) setAnalysis(res)
      })
    }
  }

  return (
    <div className="space-y-6">
      {/* Top Health Header */}
      <div className="bg-slate-900 border border-slate-800 rounded-2xl p-5 flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="flex items-center space-x-3.5">
          <div className="p-3 rounded-xl bg-purple-500/10 text-purple-400 border border-purple-500/30">
            <TrendingUp className="w-6 h-6" />
          </div>
          <div>
            <div className="flex items-center space-x-2.5">
              <h3 className="text-base font-semibold text-white">
                ABR Bitrate Ladder & BPP Quality Engine
              </h3>
              <span
                className={`text-xs px-2.5 py-0.5 rounded-full font-bold border ${
                  analysis.healthScore >= 90
                    ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                    : analysis.healthScore >= 70
                    ? 'bg-amber-500/15 text-amber-400 border-amber-500/30'
                    : 'bg-rose-500/15 text-rose-400 border-rose-500/30'
                }`}
              >
                Health: {analysis.healthScore}/100
              </span>
            </div>
            <p className="text-xs text-slate-400 mt-1">
              Dynamic Range: {analysis.dynamicRange?.toFixed(1) || '0'}x | Variants: {customRungs.length} tiers |
              BPP Formula: Bitrate / (Width × Height × FPS)
            </p>
          </div>
        </div>

        <button
          onClick={handleAddRung}
          className="inline-flex items-center space-x-1.5 px-3.5 py-2 rounded-xl text-xs font-semibold bg-sky-500/15 hover:bg-sky-500/25 text-sky-400 border border-sky-500/30 transition-colors shrink-0"
        >
          <Plus className="w-3.5 h-3.5" />
          <span>Add Ladder Tier</span>
        </button>
      </div>

      {/* Visual Bitrate Ladder Bars */}
      <div className="bg-slate-900 border border-slate-800 rounded-2xl p-5">
        <h4 className="text-xs font-semibold text-slate-300 uppercase tracking-wider mb-4">
          Bitrate Progression & Visual Spacing
        </h4>

        <div className="space-y-3.5">
          {customRungs.map((rung, idx) => {
            const barWidthPercent = Math.max(12, (rung.bitrate / maxBitrate) * 100)
            const isOptimal = rung.bppStatus === 'OPTIMAL'
            const isStarved = rung.bppStatus === 'STARVED'

            return (
              <div key={idx} className="space-y-1">
                <div className="flex items-center justify-between text-xs">
                  <div className="flex items-center space-x-2">
                    <span className="font-semibold text-white font-mono">{rung.id || rung.resolution}</span>
                    <span className="text-[11px] text-slate-400">({rung.resolution})</span>
                    <span className="text-[10px] px-1.5 py-0.5 rounded bg-slate-800 text-slate-300 font-mono">
                      {rung.codec || 'H.264'}
                    </span>
                  </div>

                  <div className="flex items-center space-x-3 text-[11px] font-mono">
                    <span className="text-sky-300 font-bold">{(rung.bitrate / 1000000).toFixed(2)} Mbps</span>
                    {rung.stepRatio > 1 && (
                      <span
                        className={`px-1.5 py-0.5 rounded text-[10px] ${
                          rung.stepRatio > 2.5
                            ? 'bg-rose-500/15 text-rose-400 border border-rose-500/30'
                            : rung.stepRatio < 1.3
                            ? 'bg-amber-500/15 text-amber-400 border border-amber-500/30'
                            : 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30'
                        }`}
                      >
                        {rung.stepRatio.toFixed(2)}x step
                      </span>
                    )}
                    <span
                      className={`px-1.5 py-0.5 rounded text-[10px] font-bold ${
                        isOptimal
                          ? 'bg-emerald-500/15 text-emerald-400'
                          : isStarved
                          ? 'bg-rose-500/15 text-rose-400'
                          : 'bg-amber-500/15 text-amber-400'
                      }`}
                    >
                      BPP: {rung.bitsPerPixel.toFixed(4)}
                    </span>
                  </div>
                </div>

                {/* Progress Bar */}
                <div className="h-3 w-full bg-slate-950 rounded-full overflow-hidden p-0.5 border border-slate-800">
                  <div
                    style={{ width: `${barWidthPercent}%` }}
                    className={`h-full rounded-full transition-all ${
                      isOptimal
                        ? 'bg-gradient-to-r from-sky-500 to-indigo-500'
                        : isStarved
                        ? 'bg-gradient-to-r from-rose-500 to-red-600'
                        : 'bg-gradient-to-r from-amber-500 to-orange-500'
                    }`}
                  />
                </div>
              </div>
            )
          })}
        </div>
      </div>

      {/* Ladder Table & Recommendations */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Table of Rungs */}
        <div className="lg:col-span-8 bg-slate-900 border border-slate-800 rounded-2xl p-5 overflow-x-auto">
          <h4 className="text-xs font-semibold text-slate-300 uppercase tracking-wider mb-3">
            Rung Parameters & Quality Breakdown
          </h4>

          <table className="w-full text-xs text-left">
            <thead>
              <tr className="border-b border-slate-800 text-[11px] text-slate-400 uppercase tracking-wider">
                <th className="py-2.5 pr-2 font-medium">Tier</th>
                <th className="py-2.5 px-2 font-medium">Resolution</th>
                <th className="py-2.5 px-2 font-medium">Bitrate (bps)</th>
                <th className="py-2.5 px-2 font-medium">FPS</th>
                <th className="py-2.5 px-2 font-medium">BPP Quality</th>
                <th className="py-2.5 pl-2 text-right font-medium">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60 font-mono">
              {customRungs.map((r, idx) => (
                <tr key={idx} className="hover:bg-slate-800/40 transition-colors">
                  <td className="py-2.5 pr-2 font-semibold text-white">{r.id}</td>
                  <td className="py-2.5 px-2">
                    <input
                      type="text"
                      value={r.resolution}
                      onChange={(e) => handleUpdateRung(idx, 'resolution', e.target.value)}
                      className="w-24 bg-slate-950 border border-slate-800 rounded px-2 py-1 text-slate-200 focus:outline-none focus:border-sky-500 font-mono text-xs"
                    />
                  </td>
                  <td className="py-2.5 px-2">
                    <input
                      type="number"
                      value={r.bitrate}
                      onChange={(e) => handleUpdateRung(idx, 'bitrate', parseInt(e.target.value, 10) || 0)}
                      className="w-28 bg-slate-950 border border-slate-800 rounded px-2 py-1 text-sky-400 font-bold focus:outline-none focus:border-sky-500 font-mono text-xs"
                    />
                  </td>
                  <td className="py-2.5 px-2">
                    <input
                      type="number"
                      value={r.fps}
                      onChange={(e) => handleUpdateRung(idx, 'fps', parseFloat(e.target.value) || 30)}
                      className="w-16 bg-slate-950 border border-slate-800 rounded px-2 py-1 text-slate-200 focus:outline-none focus:border-sky-500 font-mono text-xs"
                    />
                  </td>
                  <td className="py-2.5 px-2">
                    <span
                      className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                        r.bppStatus === 'OPTIMAL'
                          ? 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30'
                          : r.bppStatus === 'STARVED'
                          ? 'bg-rose-500/15 text-rose-400 border border-rose-500/30'
                          : 'bg-amber-500/15 text-amber-400 border border-amber-500/30'
                      }`}
                    >
                      {r.bppStatus} ({r.bitsPerPixel.toFixed(4)})
                    </span>
                  </td>
                  <td className="py-2.5 pl-2 text-right">
                    <button
                      onClick={() => handleDeleteRung(idx)}
                      disabled={customRungs.length <= 1}
                      className="p-1 text-slate-500 hover:text-rose-400 disabled:opacity-30 transition-colors"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Actionable Recommendations */}
        <div className="lg:col-span-4 bg-slate-900 border border-slate-800 rounded-2xl p-5 flex flex-col">
          <div className="flex items-center space-x-2 mb-3">
            <Sparkles className="w-4 h-4 text-sky-400" />
            <h4 className="text-xs font-semibold text-slate-300 uppercase tracking-wider">
              Optimization Advice
            </h4>
          </div>

          <div className="flex-1 space-y-3">
            {analysis.recommendations && analysis.recommendations.length > 0 ? (
              analysis.recommendations.map((rec, i) => (
                <div key={i} className="p-3 rounded-xl bg-slate-950 border border-slate-800 text-xs text-slate-300 leading-relaxed">
                  {rec}
                </div>
              ))
            ) : (
              <p className="text-xs text-slate-500">No specific optimization advice needed.</p>
            )}

            {/* Codec BPP Benchmark Reference */}
            <div className="bg-slate-950/80 rounded-xl p-3 border border-slate-800 text-[11px] text-slate-400 space-y-1.5 mt-4">
              <span className="font-semibold text-slate-200 block mb-1">Target BPP Ranges:</span>
              <div>• H.264 / AVC: 0.07 - 0.15 BPP</div>
              <div>• H.265 / HEVC: 0.04 - 0.09 BPP</div>
              <div>• AV1: 0.03 - 0.07 BPP</div>
              <div className="text-[10px] text-slate-500 pt-1">
                Values below target cause blocking artifacts; values above saturate display resolution.
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
