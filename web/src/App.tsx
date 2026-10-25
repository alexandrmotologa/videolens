import React, { useEffect, useState } from 'react'
import { Navbar } from './components/Navbar'
import { ManifestLinter } from './components/ManifestLinter'
import { AtomTreeView } from './components/AtomTreeView'
import { FastStartBanner } from './components/FastStartBanner'
import { AbrLadderChart } from './components/AbrLadderChart'
import { PlayerSandbox } from './components/PlayerSandbox'
import {
  TabType,
  SampleItem,
  HLSLintResult,
  MP4Profile,
  LadderAnalysis,
  LadderRung,
} from './types/media'
import {
  FileCode,
  Box,
  BarChart3,
  Play,
  Upload,
  RefreshCw,
  Link as LinkIcon,
  FileText,
  Loader2,
  HardDrive,
  Download,
} from 'lucide-react'

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<TabType>('linter')
  const [samples, setSamples] = useState<SampleItem[]>([])
  const [isLoading, setIsLoading] = useState(false)

  // Manifest Linter state
  const [manifestUrl, setManifestUrl] = useState('')
  const [rawManifest, setRawManifest] = useState('')
  const [inputMode, setInputMode] = useState<'url' | 'raw'>('raw')
  const [hlsResult, setHlsResult] = useState<HLSLintResult | null>(null)

  // MP4 Profiler state
  const [mp4Profile, setMp4Profile] = useState<MP4Profile | null>(null)
  const [activeMp4Sample, setActiveMp4Sample] = useState<'unoptimized' | 'faststart'>('unoptimized')

  // ABR Ladder state
  const [ladderAnalysis, setLadderAnalysis] = useState<LadderAnalysis | undefined>(undefined)

  // Fetch samples on initial load
  useEffect(() => {
    fetch('/api/samples')
      .then((res) => res.json())
      .then((data: SampleItem[]) => {
        setSamples(data)
        if (data.length > 0) {
          // Pre-load the first valid master sample
          const sample = data[0]
          if (sample.content) {
            setRawManifest(sample.content)
            runLintContent(sample.content)
          }
        }
      })
      .catch(() => {})

    // Pre-load unoptimized sample MP4 profile
    inspectMP4('unoptimized')
  }, [])

  const runLintContent = async (content: string, url = '') => {
    setIsLoading(true)
    try {
      const isDASH = content.includes('<MPD') || url.endsWith('.mpd')
      const endpoint = isDASH ? '/api/dash/lint' : '/api/hls/lint'

      const res = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content, url }),
      })
      const data = await res.json()
      if (isDASH) {
        // Adapt DASH result format into unified result
        setHlsResult({
          isMaster: true,
          version: 1,
          hasEndList: true,
          segmentCount: 0,
          discontinuityCount: 0,
          rawLines: data.rawLines || [],
          issues: data.issues || [],
          healthScore: data.healthScore,
          variants: (data.adaptationSets || []).flatMap((as: any) =>
            (as.representations || []).map((rep: any, i: number) => ({
              bandwidth: rep.bandwidth,
              resolution: rep.width ? `${rep.width}x${rep.height}` : `rep-${rep.id}`,
              width: rep.width,
              height: rep.height,
              frameRate: rep.frameRate,
              codecs: rep.codecs,
              uri: `representation-${rep.id}`,
              lineNumber: i + 1,
            }))
          ),
        })
      } else {
        setHlsResult(data)
        if (data.ladder) {
          setLadderAnalysis(data.ladder)
        }
      }
    } finally {
      setIsLoading(false)
    }
  }

  const handleLintSubmit = () => {
    if (inputMode === 'url') {
      runLintContent('', manifestUrl)
    } else {
      runLintContent(rawManifest)
    }
  }

  const inspectMP4 = async (sampleType: 'faststart' | 'unoptimized') => {
    setIsLoading(true)
    setActiveMp4Sample(sampleType)
    try {
      const res = await fetch('/api/mp4/inspect', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ sample: sampleType }),
      })
      const data = await res.json()
      setMp4Profile(data)
    } finally {
      setIsLoading(false)
    }
  }

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    setIsLoading(true)
    const formData = new FormData()
    formData.append('file', file)

    try {
      const res = await fetch('/api/mp4/inspect', {
        method: 'POST',
        body: formData,
      })
      const data = await res.json()
      setMp4Profile(data)
    } finally {
      setIsLoading(false)
    }
  }

  const handleRemediate = async () => {
    const res = await fetch('/api/mp4/remediate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ sample: activeMp4Sample }),
    })
    const blob = await res.blob()
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'videolens_faststart_remediated.mp4'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    window.URL.revokeObjectURL(url)

    // Re-inspect as faststart
    await inspectMP4('faststart')
  }

  const handleSelectSample = (sample: SampleItem) => {
    if (sample.type === 'hls_master' || sample.type === 'hls_media' || sample.type === 'dash') {
      setActiveTab('linter')
      if (sample.url) {
        setInputMode('url')
        setManifestUrl(sample.url)
        runLintContent('', sample.url)
      } else if (sample.content) {
        setInputMode('raw')
        setRawManifest(sample.content)
        runLintContent(sample.content)
      }
    }
  }

  const handleReanalyzeLadder = async (rungs: LadderRung[]) => {
    const res = await fetch('/api/ladder/analyze', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ rungs }),
    })
    return await res.json()
  }

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col">
      <Navbar
        activeTab={activeTab}
        onSelectTab={setActiveTab}
        samples={samples}
        onSelectSample={handleSelectSample}
        healthScore={
          activeTab === 'linter'
            ? hlsResult?.healthScore
            : activeTab === 'abr'
            ? ladderAnalysis?.healthScore
            : undefined
        }
      />

      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6">
        {/* TAB 1: MANIFEST LINTER */}
        {activeTab === 'linter' && (
          <div className="space-y-6">
            {/* Input Selection Bar */}
            <div className="bg-slate-900 border border-slate-800 rounded-2xl p-4 space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <div className="flex items-center space-x-1 bg-slate-950 p-1 rounded-xl border border-slate-800 text-xs">
                    <button
                      onClick={() => setInputMode('raw')}
                      className={`flex items-center space-x-1.5 px-3 py-1 rounded-lg font-medium transition-colors ${
                        inputMode === 'raw'
                          ? 'bg-slate-800 text-white shadow-sm'
                          : 'text-slate-400 hover:text-slate-200'
                      }`}
                    >
                      <FileText className="w-3.5 h-3.5" />
                      <span>Paste Manifest</span>
                    </button>
                    <button
                      onClick={() => setInputMode('url')}
                      className={`flex items-center space-x-1.5 px-3 py-1 rounded-lg font-medium transition-colors ${
                        inputMode === 'url'
                          ? 'bg-slate-800 text-white shadow-sm'
                          : 'text-slate-400 hover:text-slate-200'
                      }`}
                    >
                      <LinkIcon className="w-3.5 h-3.5" />
                      <span>Remote URL</span>
                    </button>
                  </div>
                </div>

                <button
                  onClick={handleLintSubmit}
                  disabled={isLoading}
                  className="flex items-center space-x-2 px-4 py-1.5 rounded-xl text-xs font-semibold bg-sky-500 hover:bg-sky-400 text-white shadow-sm shadow-sky-500/20 transition-colors disabled:opacity-50 cursor-pointer"
                >
                  {isLoading ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <RefreshCw className="w-4 h-4" />
                  )}
                  <span>Lint Manifest</span>
                </button>
              </div>

              {inputMode === 'url' ? (
                <input
                  type="text"
                  value={manifestUrl}
                  onChange={(e) => setManifestUrl(e.target.value)}
                  placeholder="https://example.com/stream/master.m3u8"
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-xs font-mono text-slate-200 focus:outline-none focus:border-sky-500"
                />
              ) : (
                <textarea
                  rows={6}
                  value={rawManifest}
                  onChange={(e) => setRawManifest(e.target.value)}
                  placeholder="#EXTM3U&#10;#EXT-X-VERSION:3&#10;#EXT-X-TARGETDURATION:6..."
                  className="w-full bg-slate-950 border border-slate-800 rounded-xl p-3 text-xs font-mono text-slate-200 focus:outline-none focus:border-sky-500 resize-none"
                />
              )}
            </div>

            {/* Results */}
            {hlsResult && <ManifestLinter result={hlsResult} />}
          </div>
        )}

        {/* TAB 2: MP4 ATOM PROFILER & FAST-START */}
        {activeTab === 'mp4' && (
          <div className="space-y-6">
            {/* Top Toolbar: Switch Between Unoptimized & FastStart Sample or Upload */}
            <div className="bg-slate-900 border border-slate-800 rounded-2xl p-4 flex flex-col sm:flex-row items-center justify-between gap-4">
              <div className="flex items-center space-x-2">
                <span className="text-xs text-slate-400 font-medium">Sample Generator:</span>
                <div className="flex items-center space-x-1 bg-slate-950 p-1 rounded-xl border border-slate-800 text-xs">
                  <button
                    onClick={() => inspectMP4('unoptimized')}
                    className={`px-3 py-1 rounded-lg font-medium transition-colors ${
                      activeMp4Sample === 'unoptimized'
                        ? 'bg-rose-500/20 text-rose-300 border border-rose-500/30'
                        : 'text-slate-400 hover:text-slate-200'
                    }`}
                  >
                    Unoptimized (moov at end)
                  </button>
                  <button
                    onClick={() => inspectMP4('faststart')}
                    className={`px-3 py-1 rounded-lg font-medium transition-colors ${
                      activeMp4Sample === 'faststart'
                        ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30'
                        : 'text-slate-400 hover:text-slate-200'
                    }`}
                  >
                    Fast-Start Optimized
                  </button>
                </div>
              </div>

              {/* Upload Local MP4 File */}
              <label className="flex items-center space-x-2 px-3.5 py-1.5 rounded-xl text-xs font-semibold bg-slate-800 hover:bg-slate-700/80 text-slate-200 border border-slate-700 cursor-pointer transition-colors">
                <Upload className="w-3.5 h-3.5 text-sky-400" />
                <span>Upload Custom MP4</span>
                <input
                  type="file"
                  accept="video/mp4,video/quicktime"
                  onChange={handleFileUpload}
                  className="hidden"
                />
              </label>
            </div>

            {/* Fast-Start Banner with 1-Click Remediation */}
            {mp4Profile?.fastStart && (
              <FastStartBanner
                status={mp4Profile.fastStart}
                onRemediate={handleRemediate}
              />
            )}

            {/* Atom Tree Inspector */}
            {mp4Profile && (
              <AtomTreeView
                profile={mp4Profile}
                onRemediate={handleRemediate}
              />
            )}
          </div>
        )}

        {/* TAB 3: ABR LADDER & BPP */}
        {activeTab === 'abr' && (
          <AbrLadderChart
            initialAnalysis={ladderAnalysis}
            onReanalyze={handleReanalyzeLadder}
          />
        )}

        {/* TAB 4: PLAYER & NETWORK THROTTLE SIMULATOR */}
        {activeTab === 'player' && (
          <PlayerSandbox initialUrl={manifestUrl || 'https://test-streams.mux.dev/x36xhzz/x36xhzz.m3u8'} />
        )}
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-800 bg-slate-900/50 py-6 text-center text-xs text-slate-500">
        <div className="max-w-7xl mx-auto px-4 flex flex-col sm:flex-row items-center justify-between gap-2">
          <div>
            VideoLens &bull; Modern Streaming Diagnostics Workbench &bull; RFC 8216 & ISO/IEC 14496-12 Compliant
          </div>
          <div className="font-mono text-[11px] text-slate-400">
            Go 1.23 + React 19 + Hls.js + Tailwind CSS
          </div>
        </div>
      </footer>
    </div>
  )
}
