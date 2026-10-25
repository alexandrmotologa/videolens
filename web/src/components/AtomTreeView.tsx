import React, { useState } from 'react'
import { BoxNode, MP4Profile, TrackSummary, GOPInterval } from '../types/media'
import {
  ChevronRight,
  ChevronDown,
  Box,
  Film,
  Music,
  Info,
  Layers,
  Search,
  FileDown,
  Activity,
  Sparkles,
  AlertTriangle,
  Clock,
} from 'lucide-react'

interface AtomTreeViewProps {
  profile: MP4Profile
  onRemediate?: () => Promise<void>
}

export const AtomTreeView: React.FC<AtomTreeViewProps> = ({ profile, onRemediate }) => {
  const [selectedBox, setSelectedBox] = useState<BoxNode | null>(profile.rootBoxes[0] || null)
  const [search, setSearch] = useState('')
  const [expandedKeys, setExpandedKeys] = useState<Record<string, boolean>>({
    '0-moov': true,
    '0-moov-1-trak': true,
  })
  const [selectedGOP, setSelectedGOP] = useState<GOPInterval | null>(null)

  const toggleExpand = (key: string, e: React.MouseEvent) => {
    e.stopPropagation()
    setExpandedKeys((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  const getBadgeColor = (type: string) => {
    switch (type) {
      case 'ftyp':
        return 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
      case 'moov':
        return 'bg-indigo-500/15 text-indigo-400 border-indigo-500/30'
      case 'trak':
        return 'bg-violet-500/15 text-violet-400 border-violet-500/30'
      case 'mdia':
      case 'minf':
      case 'stbl':
        return 'bg-sky-500/15 text-sky-400 border-sky-500/30'
      case 'mdat':
        return 'bg-amber-500/15 text-amber-400 border-amber-500/30'
      case 'moof':
      case 'traf':
      case 'trun':
      case 'tfdt':
      case 'sidx':
        return 'bg-fuchsia-500/15 text-fuchsia-400 border-fuchsia-500/30'
      case 'free':
      case 'skip':
        return 'bg-slate-500/15 text-slate-400 border-slate-500/30'
      default:
        return 'bg-slate-800 text-slate-300 border-slate-700'
    }
  }

  const videoTrack = profile.tracks.find((t) => t.type === 'video')

  const renderTree = (node: BoxNode, path: string, depth = 0) => {
    const hasChildren = node.children && node.children.length > 0
    const isExpanded = expandedKeys[path] || search !== ''
    const isSelected = selectedBox === node

    const matchesSearch =
      search === '' ||
      node.type.toLowerCase().includes(search.toLowerCase()) ||
      (node.children && node.children.some((c) => c.type.toLowerCase().includes(search.toLowerCase())))

    if (!matchesSearch) return null

    return (
      <div key={path} className="select-none">
        <div
          onClick={() => setSelectedBox(node)}
          style={{ paddingLeft: `${depth * 18 + 10}px` }}
          className={`flex items-center justify-between py-1.5 pr-3 text-xs font-mono rounded-lg cursor-pointer transition-colors ${
            isSelected
              ? 'bg-sky-500/20 text-white border-l-2 border-sky-400'
              : 'hover:bg-slate-800/60 text-slate-300'
          }`}
        >
          <div className="flex items-center space-x-2 truncate">
            {hasChildren ? (
              <button
                onClick={(e) => toggleExpand(path, e)}
                className="p-0.5 hover:bg-slate-700 rounded text-slate-400 hover:text-white"
              >
                {isExpanded ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />}
              </button>
            ) : (
              <span className="w-4.5 inline-block" />
            )}

            <span
              className={`px-2 py-0.5 rounded text-[11px] font-bold border ${getBadgeColor(
                node.type
              )}`}
            >
              {node.type}
            </span>
            <span className="text-slate-500 text-[10px]">@{node.offset.toLocaleString()}</span>
          </div>

          <span className="text-[10px] text-slate-400 shrink-0 font-sans">
            {(node.totalSize / 1024).toFixed(1)} KB
          </span>
        </div>

        {hasChildren && isExpanded && (
          <div className="mt-0.5">
            {node.children!.map((child, idx) =>
              renderTree(child, `${path}-${idx}-${child.type}`, depth + 1)
            )}
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Track Summaries Row */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {profile.tracks.map((trk) => (
          <div
            key={trk.trackId}
            className="bg-slate-900 border border-slate-800 rounded-xl p-4 flex items-start space-x-3.5"
          >
            <div
              className={`p-2.5 rounded-xl border mt-0.5 ${
                trk.type === 'video'
                  ? 'bg-sky-500/10 text-sky-400 border-sky-500/30'
                  : 'bg-purple-500/10 text-purple-400 border-purple-500/30'
              }`}
            >
              {trk.type === 'video' ? <Film className="w-5 h-5" /> : <Music className="w-5 h-5" />}
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold text-white uppercase tracking-wider">
                  Track #{trk.trackId} ({trk.type})
                </span>
                <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-slate-800 text-sky-300 border border-slate-700">
                  {trk.codec}
                </span>
              </div>
              <div className="mt-2 text-xs text-slate-400 space-y-1">
                {trk.width && trk.height && (
                  <div className="flex justify-between">
                    <span>Resolution:</span>
                    <span className="text-slate-200 font-mono">
                      {trk.width}x{trk.height}
                    </span>
                  </div>
                )}
                <div className="flex justify-between">
                  <span>Duration:</span>
                  <span className="text-slate-200 font-mono">{trk.durationSeconds.toFixed(2)}s</span>
                </div>
                {trk.keyframeCount ? (
                  <div className="flex justify-between">
                    <span>Keyframes (IDR):</span>
                    <span className="text-slate-200 font-mono">
                      {trk.keyframeCount} (~{trk.avgKeyframeSec?.toFixed(1)}s interval)
                    </span>
                  </div>
                ) : null}
                {trk.sampleRate ? (
                  <div className="flex justify-between">
                    <span>Sample Rate:</span>
                    <span className="text-slate-200 font-mono">{trk.sampleRate} Hz</span>
                  </div>
                ) : null}
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Visual GOP (Group of Pictures) & Keyframe Cadence Filmstrip */}
      {videoTrack?.gop && videoTrack.gop.intervals.length > 0 && (
        <div className="bg-slate-900 border border-slate-800 rounded-2xl p-5 space-y-3">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div className="flex items-center space-x-2.5">
              <Activity className="w-4 h-4 text-sky-400" />
              <h4 className="text-xs font-semibold text-slate-200 uppercase tracking-wider">
                GOP Keyframe Cadence Visualizer
              </h4>
              <span
                className={`text-[11px] font-bold px-2 py-0.5 rounded-full border ${
                  videoTrack.gop.isFixedGOP
                    ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
                    : 'bg-amber-500/15 text-amber-400 border-amber-500/30'
                }`}
              >
                {videoTrack.gop.cadenceType} GOP (~{videoTrack.gop.meanGOPDuration.toFixed(2)}s)
              </span>
            </div>

            <div className="text-xs text-slate-400 font-mono">
              {videoTrack.gop.totalKeyframes} Keyframes | Min: {videoTrack.gop.minGOPDuration.toFixed(2)}s | Max: {videoTrack.gop.maxGOPDuration.toFixed(2)}s
            </div>
          </div>

          <p className="text-xs text-slate-400 leading-relaxed">
            {videoTrack.gop.recommendation}
          </p>

          {/* Interactive GOP Intervals Strip */}
          <div className="flex items-center space-x-2 overflow-x-auto pt-2 pb-1.5 scrollbar-thin">
            {videoTrack.gop.intervals.map((gop) => {
              const isSelected = selectedGOP === gop
              return (
                <button
                  key={gop.index}
                  onClick={() => setSelectedGOP(gop)}
                  className={`shrink-0 w-20 p-2.5 rounded-xl text-center border transition-all cursor-pointer ${
                    isSelected
                      ? 'bg-sky-500/20 border-sky-400 ring-2 ring-sky-500/40 text-white'
                      : 'bg-slate-950 border-slate-800 text-slate-300 hover:border-slate-700'
                  }`}
                >
                  <div className="text-[10px] text-slate-500 font-mono">GOP #{gop.index}</div>
                  <div className="text-xs font-bold font-mono text-sky-400 mt-0.5">
                    {gop.durationSeconds.toFixed(2)}s
                  </div>
                  <div className="text-[9px] text-slate-400 mt-0.5">
                    {gop.durationFrames} frames
                  </div>
                </button>
              )
            })}
          </div>

          {selectedGOP && (
            <div className="bg-slate-950 p-3 rounded-xl border border-slate-800 text-xs text-slate-300 flex flex-col sm:flex-row sm:items-center justify-between gap-1.5 sm:gap-2 font-mono">
              <span className="font-semibold text-white">GOP #{selectedGOP.index} Selected</span>
              <span>Start: Frame {selectedGOP.startFrame} (~{selectedGOP.startSeconds.toFixed(2)}s)</span>
              <span className="text-sky-400 font-bold">Span: {selectedGOP.durationSeconds.toFixed(3)}s</span>
            </div>
          )}
        </div>
      )}

      {/* Main Split View: Atom Tree (Left) vs Atom Inspector (Right) */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Tree Column */}
        <div className="lg:col-span-6 bg-slate-900 border border-slate-800 rounded-2xl p-4 flex flex-col h-[560px]">
          <div className="flex items-center justify-between mb-3 pb-3 border-b border-slate-800">
            <div className="flex items-center space-x-2">
              <Layers className="w-4 h-4 text-sky-400" />
              <h4 className="text-sm font-semibold text-white">ISO Box Hierarchy</h4>
            </div>
            <div className="relative">
              <Search className="w-3.5 h-3.5 text-slate-400 absolute left-2.5 top-2.5" />
              <input
                type="text"
                placeholder="Filter atom..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-8 pr-3 py-1 text-xs rounded-lg bg-slate-950 border border-slate-800 text-slate-200 focus:outline-none focus:border-sky-500 w-36 sm:w-44"
              />
            </div>
          </div>

          <div className="flex-1 overflow-y-auto space-y-0.5 pr-1">
            {profile.rootBoxes.map((box, idx) =>
              renderTree(box, `${idx}-${box.type}`)
            )}
          </div>
        </div>

        {/* Inspector Column */}
        <div className="lg:col-span-6 bg-slate-900 border border-slate-800 rounded-2xl p-5 flex flex-col h-[560px]">
          <div className="flex items-center justify-between mb-4 pb-3 border-b border-slate-800">
            <div className="flex items-center space-x-2">
              <Info className="w-4 h-4 text-indigo-400" />
              <h4 className="text-sm font-semibold text-white">Atom Inspector</h4>
            </div>
            {selectedBox && (
              <span
                className={`px-2.5 py-0.5 rounded text-xs font-mono font-bold border ${getBadgeColor(
                  selectedBox.type
                )}`}
              >
                {selectedBox.type}
              </span>
            )}
          </div>

          {selectedBox ? (
            <div className="flex-1 overflow-y-auto space-y-4">
              {/* Core Box Fields */}
              <div className="bg-slate-950 rounded-xl p-4 border border-slate-800 space-y-2 text-xs">
                <div className="flex justify-between py-1 border-b border-slate-800/60">
                  <span className="text-slate-400">FourCC Identifier:</span>
                  <span className="text-white font-mono font-bold">{selectedBox.type}</span>
                </div>
                <div className="flex justify-between py-1 border-b border-slate-800/60">
                  <span className="text-slate-400">Absolute File Offset:</span>
                  <span className="text-sky-400 font-mono">
                    {selectedBox.offset.toLocaleString()} bytes (0x{selectedBox.offset.toString(16)})
                  </span>
                </div>
                <div className="flex justify-between py-1 border-b border-slate-800/60">
                  <span className="text-slate-400">Total Box Size:</span>
                  <span className="text-white font-mono">
                    {selectedBox.totalSize.toLocaleString()} bytes ({(selectedBox.totalSize / 1024).toFixed(2)} KB)
                  </span>
                </div>
                <div className="flex justify-between py-1 border-b border-slate-800/60">
                  <span className="text-slate-400">Header Size:</span>
                  <span className="text-white font-mono">{selectedBox.headerSize} bytes</span>
                </div>
                <div className="flex justify-between py-1">
                  <span className="text-slate-400">Payload Size:</span>
                  <span className="text-white font-mono">
                    {selectedBox.payloadSize.toLocaleString()} bytes
                  </span>
                </div>
              </div>

              {/* Decoded Details */}
              {selectedBox.details && Object.keys(selectedBox.details).length > 0 && (
                <div>
                  <h5 className="text-xs font-semibold text-slate-300 mb-2 uppercase tracking-wider">
                    Decoded Box Properties
                  </h5>
                  <div className="bg-slate-950 rounded-xl p-4 border border-slate-800 space-y-2 text-xs font-mono">
                    {Object.entries(selectedBox.details).map(([key, val]) => (
                      <div key={key} className="flex justify-between py-1 border-b border-slate-800/40 last:border-none">
                        <span className="text-slate-400 font-sans">{key}:</span>
                        <span className="text-sky-300 truncate max-w-xs text-right">
                          {Array.isArray(val) ? val.join(', ') : String(val)}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Explanation Note */}
              <div className="bg-slate-800/40 rounded-xl p-3.5 border border-slate-800 text-xs text-slate-400 leading-relaxed">
                <span className="font-semibold text-slate-200">ISO/IEC 14496-12 Reference:</span>{' '}
                {selectedBox.type === 'moov' &&
                  'Container box storing all video and audio metadata, track definitions, timescales, and sample tables.'}
                {selectedBox.type === 'mdat' &&
                  'Container holding raw media samples (video frames and audio packets). In Fast-Start files, mdat appears after moov.'}
                {selectedBox.type === 'trak' &&
                  'Individual track container representing a video, audio, or subtitle stream.'}
                {selectedBox.type === 'stbl' &&
                  'Sample table box detailing sample timing, chunk offsets, keyframes, and chunk mappings.'}
                {selectedBox.type === 'stco' &&
                  '32-bit chunk offset table. For Fast-Start remediation, all offsets inside stco are shifted forward by moov size.'}
                {selectedBox.type === 'stss' &&
                  'Sync sample table specifying keyframe (IDR) indexes for random access and ABR variant switching.'}
                {selectedBox.type === 'moof' &&
                  'Movie fragment container (CMAF / fMP4) defining independent streaming chunk headers.'}
                {selectedBox.type === 'tfdt' &&
                  'Track fragment decode time box specifying the exact presentation timestamp of this media chunk.'}
                {selectedBox.type === 'sidx' &&
                  'Segment index box allowing subsegment seekability and random access indexing in DASH & CMAF streams.'}
              </div>
            </div>
          ) : (
            <div className="flex-1 flex items-center justify-center text-xs text-slate-500">
              Select an atom box from the hierarchy on the left to inspect its binary attributes.
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
