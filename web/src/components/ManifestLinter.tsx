import React, { useState } from 'react'
import { HLSLintResult, LintIssue } from '../types/media'
import {
  AlertTriangle,
  XCircle,
  Info,
  CheckCircle2,
  Clock,
  Layers,
  Activity,
  ArrowRight,
  Filter,
  Flame,
} from 'lucide-react'

interface ManifestLinterProps {
  result: HLSLintResult
  onInspectVariant?: (uri: string) => void
}

export const ManifestLinter: React.FC<ManifestLinterProps> = ({ result, onInspectVariant }) => {
  const [filterSeverity, setFilterSeverity] = useState<'ALL' | 'ERROR' | 'WARNING'>('ALL')
  const [selectedLine, setSelectedLine] = useState<number | null>(null)

  const filteredIssues = result.issues.filter((issue) => {
    if (filterSeverity === 'ALL') return true
    return issue.severity === filterSeverity
  })

  // Map issues by line number for inline badge rendering
  const issuesByLine = new Map<number, LintIssue[]>()
  for (const issue of result.issues) {
    const list = issuesByLine.get(issue.line) || []
    list.push(issue)
    issuesByLine.set(issue.line, list)
  }

  const getTagColor = (line: string) => {
    if (line.startsWith('#EXT-X-STREAM-INF')) return 'text-purple-400 font-semibold'
    if (line.startsWith('#EXT-X-MEDIA')) return 'text-violet-400'
    if (line.startsWith('#EXT-X-TARGETDURATION')) return 'text-amber-400 font-semibold'
    if (line.startsWith('#EXT-X-DISCONTINUITY')) return 'text-rose-400 font-bold bg-rose-500/10 px-1 rounded'
    if (line.startsWith('#EXTINF')) return 'text-sky-400'
    if (line.startsWith('#EXT-X-ENDLIST')) return 'text-emerald-400 font-bold'
    if (line.startsWith('#EXT')) return 'text-slate-300'
    if (line.startsWith('<')) return 'text-indigo-300'
    return 'text-slate-400'
  }

  return (
    <div className="space-y-6">
      {/* Overview Stat Cards */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3.5">
        <div className="bg-slate-900 border border-slate-800 rounded-xl p-3.5">
          <div className="flex items-center space-x-1.5 text-xs text-slate-400">
            <Activity className="w-3.5 h-3.5 text-sky-400" />
            <span>Health Score</span>
          </div>
          <div
            className={`text-2xl font-bold font-mono mt-1 ${
              result.healthScore >= 90
                ? 'text-emerald-400'
                : result.healthScore >= 70
                ? 'text-amber-400'
                : 'text-rose-400'
            }`}
          >
            {result.healthScore}/100
          </div>
          <div className="text-[10px] text-slate-500 mt-0.5">
            {result.issues.length} {result.issues.length === 1 ? 'finding' : 'findings'}
          </div>
        </div>

        <div className="bg-slate-900 border border-slate-800 rounded-xl p-3.5">
          <div className="flex items-center space-x-1.5 text-xs text-slate-400">
            <Layers className="w-3.5 h-3.5 text-indigo-400" />
            <span>Playlist Type</span>
          </div>
          <div className="text-xl font-bold text-white mt-1">
            {result.isMaster ? 'Master' : result.playlistType || 'Media (VOD)'}
          </div>
          <div className="text-[10px] text-slate-500 mt-0.5">
            {result.isMaster
              ? `${result.variants?.length || 0} variant streams`
              : `${result.segmentCount} segments`}
          </div>
        </div>

        {!result.isMaster && (
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-3.5">
            <div className="flex items-center space-x-1.5 text-xs text-slate-400">
              <Clock className="w-3.5 h-3.5 text-purple-400" />
              <span>Target Duration</span>
            </div>
            <div className="text-xl font-bold font-mono text-white mt-1">
              {result.targetDuration ? `${result.targetDuration}s` : 'N/A'}
            </div>
            <div className="text-[10px] text-slate-500 mt-0.5">
              Total: {result.totalDuration ? result.totalDuration.toFixed(1) : 0}s
            </div>
          </div>
        )}

        {!result.isMaster && result.durationMetrics && (
          <div className="bg-slate-900 border border-slate-800 rounded-xl p-3.5">
            <div className="flex items-center space-x-1.5 text-xs text-slate-400">
              <Flame className="w-3.5 h-3.5 text-amber-400" />
              <span>Segment Drift</span>
            </div>
            <div
              className={`text-xl font-bold font-mono mt-1 ${
                result.durationMetrics.driftPercent > 15 ? 'text-amber-400' : 'text-emerald-400'
              }`}
            >
              {result.durationMetrics.driftPercent.toFixed(1)}%
            </div>
            <div className="text-[10px] text-slate-500 mt-0.5">
              StdDev: {result.durationMetrics.stdDev.toFixed(2)}s
            </div>
          </div>
        )}

        <div className="bg-slate-900 border border-slate-800 rounded-xl p-3.5">
          <div className="flex items-center space-x-1.5 text-xs text-slate-400">
            <AlertTriangle className="w-3.5 h-3.5 text-rose-400" />
            <span>Discontinuities</span>
          </div>
          <div className="text-xl font-bold font-mono text-white mt-1">
            {result.discontinuityCount}
          </div>
          <div className="text-[10px] text-slate-500 mt-0.5">
            {result.hasEndList ? 'Terminated with #EXT-X-ENDLIST' : 'Live sliding window'}
          </div>
        </div>
      </div>

      {/* Segment Durations Waterfall (if media playlist) */}
      {!result.isMaster && result.segments && result.segments.length > 0 && (
        <div className="bg-slate-900 border border-slate-800 rounded-2xl p-4">
          <div className="flex items-center justify-between mb-2">
            <h4 className="text-xs font-semibold text-slate-300 uppercase tracking-wider">
              Segment Duration Histogram & Drift
            </h4>
            <span className="text-[11px] text-slate-400 font-mono">
              Target: {result.targetDuration}s
            </span>
          </div>

          <div className="flex items-end space-x-1 h-20 pt-2 overflow-x-auto">
            {result.segments.map((seg) => {
              const maxScale = (result.targetDuration || 6) * 1.3
              const heightPercent = Math.min(100, (seg.duration / maxScale) * 100)
              const isExceeded = result.targetDuration && seg.duration > result.targetDuration

              return (
                <div
                  key={seg.index}
                  onClick={() => setSelectedLine(seg.lineNumber)}
                  title={`Segment #${seg.index}: ${seg.duration.toFixed(3)}s (Line ${seg.lineNumber})`}
                  className="flex-1 min-w-[12px] group relative cursor-pointer"
                >
                  <div
                    style={{ height: `${heightPercent}%` }}
                    className={`rounded-t transition-all ${
                      isExceeded
                        ? 'bg-rose-500 group-hover:bg-rose-400'
                        : seg.discontinuity
                        ? 'bg-amber-400 group-hover:bg-amber-300'
                        : 'bg-sky-500/80 group-hover:bg-sky-400'
                    }`}
                  />
                  {seg.discontinuity && (
                    <div className="w-1.5 h-1.5 rounded-full bg-rose-400 mx-auto mt-1" />
                  )}
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* Main Split: Lint Findings (Left) vs Raw Syntax View (Right) */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Issues List */}
        <div className="lg:col-span-5 bg-slate-900 border border-slate-800 rounded-2xl p-4 flex flex-col h-[600px]">
          <div className="flex items-center justify-between mb-3 pb-3 border-b border-slate-800">
            <div className="flex items-center space-x-2">
              <AlertTriangle className="w-4 h-4 text-amber-400" />
              <h4 className="text-sm font-semibold text-white">Validation Findings</h4>
              <span className="text-xs px-2 py-0.5 rounded-full bg-slate-800 text-slate-300 border border-slate-700 font-mono">
                {result.issues.length}
              </span>
            </div>

            {/* Filter Tabs */}
            <div className="flex items-center space-x-1 bg-slate-950 p-0.5 rounded-lg border border-slate-800 text-[11px]">
              {(['ALL', 'ERROR', 'WARNING'] as const).map((s) => (
                <button
                  key={s}
                  onClick={() => setFilterSeverity(s)}
                  className={`px-2 py-0.5 rounded font-medium transition-colors ${
                    filterSeverity === s
                      ? 'bg-slate-800 text-white'
                      : 'text-slate-400 hover:text-slate-200'
                  }`}
                >
                  {s}
                </button>
              ))}
            </div>
          </div>

          <div className="flex-1 overflow-y-auto space-y-2.5 pr-1">
            {filteredIssues.length === 0 ? (
              <div className="h-full flex flex-col items-center justify-center text-center p-6 text-slate-500">
                <CheckCircle2 className="w-8 h-8 text-emerald-400 mb-2 opacity-80" />
                <p className="text-xs font-medium text-slate-300">No issues found</p>
                <p className="text-[11px] text-slate-500 mt-1">
                  Playlist adheres strictly to streaming specification rules.
                </p>
              </div>
            ) : (
              filteredIssues.map((issue, idx) => {
                const isError = issue.severity === 'ERROR'
                const isWarn = issue.severity === 'WARNING'

                return (
                  <div
                    key={idx}
                    onClick={() => setSelectedLine(issue.line)}
                    className={`p-3.5 rounded-xl border text-xs cursor-pointer transition-all ${
                      selectedLine === issue.line
                        ? 'ring-2 ring-sky-500 border-transparent bg-slate-800'
                        : isError
                        ? 'bg-rose-950/20 border-rose-500/30 hover:border-rose-500/50'
                        : isWarn
                        ? 'bg-amber-950/20 border-amber-500/30 hover:border-amber-500/50'
                        : 'bg-slate-950 border-slate-800 hover:border-slate-700'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <span
                        className={`px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider border ${
                          isError
                            ? 'bg-rose-500/15 text-rose-400 border-rose-500/30'
                            : isWarn
                            ? 'bg-amber-500/15 text-amber-400 border-amber-500/30'
                            : 'bg-sky-500/15 text-sky-400 border-sky-500/30'
                        }`}
                      >
                        {issue.severity}
                      </span>
                      <span className="text-[11px] font-mono text-slate-400">
                        Line {issue.line}
                      </span>
                    </div>

                    <p className="font-semibold text-slate-200 mt-2">{issue.message}</p>
                    {issue.recommendation && (
                      <p className="text-[11px] text-slate-400 mt-1 leading-relaxed">
                        <span className="text-sky-400 font-medium">Fix:</span> {issue.recommendation}
                      </p>
                    )}
                  </div>
                )
              })
            )}
          </div>
        </div>

        {/* Syntax Viewer with Line Numbers */}
        <div className="lg:col-span-7 bg-slate-950 border border-slate-800 rounded-2xl p-4 flex flex-col h-[600px]">
          <div className="flex items-center justify-between mb-3 pb-3 border-b border-slate-800">
            <h4 className="text-sm font-semibold text-white font-mono">Raw Manifest Text</h4>
            <span className="text-xs text-slate-500 font-mono">
              {result.rawLines.length} lines
            </span>
          </div>

          <div className="flex-1 overflow-y-auto font-mono text-xs space-y-0.5 pr-2">
            {result.rawLines.map((line, idx) => {
              const lineNum = idx + 1
              const lineIssues = issuesByLine.get(lineNum)
              const isSelected = selectedLine === lineNum

              return (
                <div
                  key={lineNum}
                  id={`manifest-line-${lineNum}`}
                  className={`flex items-start group py-0.5 px-2 rounded transition-colors ${
                    isSelected
                      ? 'bg-sky-500/20 text-white'
                      : lineIssues && lineIssues.some((i) => i.severity === 'ERROR')
                      ? 'bg-rose-950/30'
                      : lineIssues
                      ? 'bg-amber-950/20'
                      : 'hover:bg-slate-900/60'
                  }`}
                >
                  <span className="w-8 shrink-0 text-slate-600 select-none text-right pr-3 text-[11px]">
                    {lineNum}
                  </span>
                  <div className="flex-1 break-all">
                    <span className={getTagColor(line)}>{line || ' '}</span>
                    {lineIssues && (
                      <div className="mt-1 space-y-1">
                        {lineIssues.map((li, i) => (
                          <div
                            key={i}
                            className={`inline-flex items-center space-x-1 px-2 py-0.5 rounded text-[10px] mr-2 ${
                              li.severity === 'ERROR'
                                ? 'bg-rose-500/20 text-rose-300 border border-rose-500/30'
                                : 'bg-amber-500/20 text-amber-300 border border-amber-500/30'
                            }`}
                          >
                            <AlertTriangle className="w-3 h-3 shrink-0" />
                            <span>{li.message}</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      </div>
    </div>
  )
}
