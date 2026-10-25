import React from 'react'
import { TabType, SampleItem } from '../types/media'
import { FileCode, Box, BarChart3, Play, ChevronDown, CheckCircle2, AlertTriangle, XCircle } from 'lucide-react'

interface NavbarProps {
  activeTab: TabType
  onSelectTab: (tab: TabType) => void
  samples: SampleItem[]
  onSelectSample: (sample: SampleItem) => void
  healthScore?: number
}

export const Navbar: React.FC<NavbarProps> = ({
  activeTab,
  onSelectTab,
  samples,
  onSelectSample,
  healthScore,
}) => {
  const tabs = [
    { id: 'linter' as TabType, label: 'Manifest Linter', shortLabel: 'Linter', icon: FileCode },
    { id: 'mp4' as TabType, label: 'MP4 Box Profiler', shortLabel: 'MP4', icon: Box },
    { id: 'abr' as TabType, label: 'ABR Ladder & BPP', shortLabel: 'ABR', icon: BarChart3 },
    { id: 'player' as TabType, label: 'Player & Throttler', shortLabel: 'Player', icon: Play },
  ]

  return (
    <header className="border-b border-slate-800 bg-slate-900/95 backdrop-blur sticky top-0 z-50">
      <div className="max-w-7xl mx-auto px-3.5 sm:px-6 lg:px-8">
        {/* Top Bar */}
        <div className="flex items-center justify-between h-14 sm:h-16">
          {/* Logo & Brand */}
          <div className="flex items-center space-x-2.5 sm:space-x-3 shrink-0">
            <div className="relative flex items-center justify-center w-8 h-8 sm:w-10 sm:h-10 rounded-xl bg-gradient-to-tr from-sky-500 via-indigo-500 to-purple-500 p-0.5 shadow-md shadow-sky-500/20 shrink-0">
              <div className="w-full h-full bg-slate-950 rounded-[10px] flex items-center justify-center">
                <img src="/logo.svg" alt="VideoLens" className="w-5 h-5 sm:w-7 sm:h-7" />
              </div>
            </div>
            <div>
              <div className="flex items-center space-x-1.5 sm:space-x-2">
                <span className="text-base sm:text-lg font-bold tracking-tight text-white">VideoLens</span>
                <span className="text-[9px] sm:text-[10px] font-semibold px-1.5 sm:px-2 py-0.5 rounded-full bg-sky-500/10 text-sky-400 border border-sky-500/20">
                  v1.0
                </span>
              </div>
              <p className="text-[11px] text-slate-400 hidden lg:block">Streaming Media Diagnostics Workbench</p>
            </div>
          </div>

          {/* Desktop Navigation Tabs (Inline on Large screens) */}
          <nav className="hidden lg:flex items-center space-x-1.5">
            {tabs.map((tab) => {
              const Icon = tab.icon
              const isActive = activeTab === tab.id
              return (
                <button
                  key={tab.id}
                  onClick={() => onSelectTab(tab.id)}
                  className={`flex items-center space-x-2 px-3 py-1.5 rounded-xl text-xs font-medium transition-all cursor-pointer ${
                    isActive
                      ? 'bg-sky-500/15 text-sky-400 border border-sky-500/30 shadow-sm shadow-sky-500/10'
                      : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60 border border-transparent'
                  }`}
                >
                  <Icon className={`w-3.5 h-3.5 ${isActive ? 'text-sky-400' : 'text-slate-400'}`} />
                  <span>{tab.label}</span>
                </button>
              )
            })}
          </nav>

          {/* Right Controls: Sample Selector & Health Badge */}
          <div className="flex items-center space-x-2 sm:space-x-3 shrink-0">
            {/* Quick Sample Selector */}
            <div className="relative group">
              <button className="flex items-center space-x-1.5 px-2.5 sm:px-3 py-1.5 rounded-xl text-xs font-medium bg-slate-800 text-slate-300 border border-slate-700 hover:bg-slate-700/80 transition-colors cursor-pointer">
                <span>Samples</span>
                <ChevronDown className="w-3.5 h-3.5 text-slate-400" />
              </button>
              <div className="absolute right-0 mt-2 w-64 rounded-xl bg-slate-900 border border-slate-700 shadow-xl py-1 hidden group-hover:block z-50">
                <div className="px-3 py-1.5 text-[11px] font-semibold text-slate-400 uppercase tracking-wider border-b border-slate-800">
                  Test Fixtures
                </div>
                {samples.map((s) => (
                  <button
                    key={s.id}
                    onClick={() => onSelectSample(s)}
                    className="w-full text-left px-3 py-2 text-xs text-slate-300 hover:bg-slate-800 hover:text-white transition-colors cursor-pointer"
                  >
                    <div className="font-medium truncate">{s.name}</div>
                    <div className="text-[10px] text-slate-400 truncate">{s.description}</div>
                  </button>
                ))}
              </div>
            </div>

            {/* Health Score Pill */}
            {typeof healthScore === 'number' && (
              <div
                className={`flex items-center space-x-1.5 px-2.5 py-1 rounded-full text-xs font-semibold border shrink-0 ${
                  healthScore >= 90
                    ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                    : healthScore >= 70
                    ? 'bg-amber-500/10 text-amber-400 border-amber-500/30'
                    : 'bg-rose-500/10 text-rose-400 border-rose-500/30'
                }`}
              >
                {healthScore >= 90 ? (
                  <CheckCircle2 className="w-3.5 h-3.5 shrink-0" />
                ) : healthScore >= 70 ? (
                  <AlertTriangle className="w-3.5 h-3.5 shrink-0" />
                ) : (
                  <XCircle className="w-3.5 h-3.5 shrink-0" />
                )}
                <span className="font-mono">{healthScore}/100</span>
              </div>
            )}
          </div>
        </div>

        {/* Mobile & Tablet Segmented Tab Bar (Visible on < lg screens) */}
        <nav className="grid grid-cols-4 gap-1 py-1.5 border-t border-slate-800/60 lg:hidden">
          {tabs.map((tab) => {
            const Icon = tab.icon
            const isActive = activeTab === tab.id
            return (
              <button
                key={tab.id}
                onClick={() => onSelectTab(tab.id)}
                className={`flex items-center justify-center space-x-1.5 py-2 px-1 rounded-xl text-xs font-medium transition-all cursor-pointer ${
                  isActive
                    ? 'bg-sky-500/20 text-sky-300 border border-sky-500/30 font-semibold shadow-sm'
                    : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/40 border border-transparent'
                }`}
              >
                <Icon className={`w-3.5 h-3.5 shrink-0 ${isActive ? 'text-sky-400' : 'text-slate-400'}`} />
                <span className="truncate text-[11px]">{tab.shortLabel}</span>
              </button>
            )
          })}
        </nav>
      </div>
    </header>
  )
}
