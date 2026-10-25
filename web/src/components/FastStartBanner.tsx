import React, { useState } from 'react'
import { FastStartStatus } from '../types/media'
import { Zap, AlertCircle, CheckCircle2, Download, Loader2, Clock, HardDrive, ArrowRight } from 'lucide-react'

interface FastStartBannerProps {
  status: FastStartStatus
  onRemediate?: () => Promise<void>
}

export const FastStartBanner: React.FC<FastStartBannerProps> = ({ status, onRemediate }) => {
  const [remediating, setRemediating] = useState(false)

  const handleFix = async () => {
    if (!onRemediate) return
    setRemediating(true)
    try {
      await onRemediate()
    } finally {
      setRemediating(false)
    }
  }

  const isOptimized = status.isFastStart

  return (
    <div
      className={`rounded-2xl border p-5 transition-all ${
        isOptimized
          ? 'bg-emerald-950/20 border-emerald-500/30'
          : 'bg-rose-950/20 border-rose-500/30'
      }`}
    >
      <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
        {/* Left Status Info */}
        <div className="flex items-start space-x-3.5">
          <div
            className={`p-2.5 rounded-xl border mt-0.5 ${
              isOptimized
                ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                : 'bg-rose-500/10 text-rose-400 border-rose-500/30'
            }`}
          >
            {isOptimized ? <CheckCircle2 className="w-5 h-5" /> : <AlertCircle className="w-5 h-5" />}
          </div>
          <div>
            <div className="flex items-center space-x-2.5">
              <h3 className="text-base font-semibold text-white">
                Fast-Start Web Streaming Optimization
              </h3>
              <span
                className={`text-xs px-2.5 py-0.5 rounded-full font-bold uppercase tracking-wider border ${
                  isOptimized
                    ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/40'
                    : 'bg-rose-500/15 text-rose-400 border-rose-500/40'
                }`}
              >
                {status.statusBadge}
              </span>
            </div>
            <p className="text-xs text-slate-300 mt-1 max-w-2xl leading-relaxed">
              {status.message}
            </p>
          </div>
        </div>

        {/* Right Action Button */}
        {!isOptimized && onRemediate && (
          <button
            onClick={handleFix}
            disabled={remediating}
            className="inline-flex items-center space-x-2 px-4 py-2.5 rounded-xl text-xs font-semibold bg-gradient-to-r from-sky-500 to-indigo-600 hover:from-sky-400 hover:to-indigo-500 text-white shadow-lg shadow-sky-500/25 transition-all disabled:opacity-50 shrink-0 cursor-pointer"
          >
            {remediating ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Download className="w-4 h-4" />
            )}
            <span>{remediating ? 'Relocating Atoms...' : 'Download Fast-Start Fixed MP4'}</span>
          </button>
        )}
      </div>

      {/* Metrics Row */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-4 pt-4 border-t border-slate-800/80">
        <div className="bg-slate-900/60 rounded-xl p-3 border border-slate-800">
          <div className="flex items-center space-x-1.5 text-[11px] font-medium text-slate-400">
            <HardDrive className="w-3.5 h-3.5 text-sky-400" />
            <span>Metadata (moov)</span>
          </div>
          <div className="text-sm font-semibold font-mono text-white mt-1">
            Byte {status.moovOffset.toLocaleString()}
          </div>
          <div className="text-[10px] text-slate-500 mt-0.5">
            {status.moovPositionPercent.toFixed(1)}% into file
          </div>
        </div>

        <div className="bg-slate-900/60 rounded-xl p-3 border border-slate-800">
          <div className="flex items-center space-x-1.5 text-[11px] font-medium text-slate-400">
            <HardDrive className="w-3.5 h-3.5 text-indigo-400" />
            <span>Media Data (mdat)</span>
          </div>
          <div className="text-sm font-semibold font-mono text-white mt-1">
            Byte {status.mdatOffset.toLocaleString()}
          </div>
          <div className="text-[10px] text-slate-500 mt-0.5">
            Size: {(status.mdatSize / 1024).toFixed(1)} KB
          </div>
        </div>

        <div className="bg-slate-900/60 rounded-xl p-3 border border-slate-800">
          <div className="flex items-center space-x-1.5 text-[11px] font-medium text-slate-400">
            <Clock className="w-3.5 h-3.5 text-amber-400" />
            <span>Startup Latency</span>
          </div>
          <div className="text-sm font-semibold font-mono text-white mt-1">
            {status.isFastStart ? '0 ms (Instant)' : `~${status.initialLatencyEstimateMs.toFixed(0)} ms`}
          </div>
          <div className="text-[10px] text-slate-500 mt-0.5">@ 10 Mbps connection</div>
        </div>

        <div className="bg-slate-900/60 rounded-xl p-3 border border-slate-800">
          <div className="flex items-center space-x-1.5 text-[11px] font-medium text-slate-400">
            <Zap className="w-3.5 h-3.5 text-purple-400" />
            <span>Atom Order</span>
          </div>
          <div className="flex items-center space-x-1.5 text-xs font-mono text-white mt-1.5">
            <span className={status.isFastStart ? 'text-sky-400 font-bold' : 'text-slate-400'}>
              {status.isFastStart ? 'moov' : 'mdat'}
            </span>
            <ArrowRight className="w-3 h-3 text-slate-500" />
            <span className={status.isFastStart ? 'text-slate-400' : 'text-rose-400 font-bold'}>
              {status.isFastStart ? 'mdat' : 'moov'}
            </span>
          </div>
          <div className="text-[10px] text-slate-500 mt-0.5">
            {status.isFastStart ? 'Web-ready structure' : 'Delayed playback'}
          </div>
        </div>
      </div>
    </div>
  )
}
