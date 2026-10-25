import React, { useEffect, useRef, useState } from 'react'
import Hls from 'hls.js'
import { ThrottleControl, ThrottleConfig, THROTTLE_PRESETS } from './ThrottleControl'
import {
  Play,
  Pause,
  RotateCcw,
  Activity,
  AlertTriangle,
  Clock,
  Gauge,
  Film,
  Zap,
  Radio,
  CheckCircle2,
} from 'lucide-react'

interface PlayerSandboxProps {
  initialUrl?: string
}

export const PlayerSandbox: React.FC<PlayerSandboxProps> = ({
  initialUrl = 'https://test-streams.mux.dev/x36xhzz/x36xhzz.m3u8',
}) => {
  const videoRef = useRef<HTMLVideoElement>(null)
  const hlsRef = useRef<Hls | null>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)

  const [streamUrl, setStreamUrl] = useState(initialUrl)
  const [isPlaying, setIsPlaying] = useState(false)
  const [currentTime, setCurrentTime] = useState(0)
  const [duration, setDuration] = useState(0)
  const [forwardBuffer, setForwardBuffer] = useState(0)
  const [activeLevel, setActiveLevel] = useState<string>('Auto')
  const [activeBitrate, setActiveBitrate] = useState<number>(0)
  const [droppedFrames, setDroppedFrames] = useState(0)
  const [stallCount, setStallCount] = useState(0)
  const [stallDurationSec, setStallDurationSec] = useState(0)
  const [downloadSpeedKbps, setDownloadSpeedKbps] = useState(0)
  const [eventLogs, setEventLogs] = useState<{ time: string; text: string; type: string }[]>([])

  const [throttleConfig, setThrottleConfig] = useState<ThrottleConfig>(THROTTLE_PRESETS[0])

  // Refs for tracking stall durations
  const stallStartRef = useRef<number | null>(null)
  const throttleRef = useRef<ThrottleConfig>(throttleConfig)
  throttleRef.current = throttleConfig

  const addLog = (text: string, type = 'info') => {
    const time = new Date().toLocaleTimeString()
    setEventLogs((prev) => [{ time, text, type }, ...prev.slice(0, 49)])
  }

  // Load stream
  const loadStream = (url: string) => {
    const video = videoRef.current
    if (!video) return

    if (hlsRef.current) {
      hlsRef.current.destroy()
      hlsRef.current = null
    }

    setStallCount(0)
    setStallDurationSec(0)
    setDroppedFrames(0)
    setForwardBuffer(0)
    addLog(`Loading stream: ${url}`, 'info')

    // Check HLS support
    if (Hls.isSupported()) {
      // Custom Hls.js configuration with custom fragment loader for network throttling
      class ThrottledLoader extends Hls.DefaultConfig.loader {
        load(context: any, config: any, callbacks: any) {
          const cfg = throttleRef.current
          const startTime = performance.now()

          const originalOnSuccess = callbacks.onSuccess
          callbacks.onSuccess = (response: any, stats: any, ctx: any, networkDetails: any) => {
            const dataLen = response.data ? (response.data.byteLength || response.data.length || 0) : 0
            const elapsedMs = performance.now() - startTime

            // Calculate simulated bandwidth delay if throttling enabled
            let artificialDelayMs = cfg.latencyMs
            if (cfg.bandwidthKbps > 0 && dataLen > 0) {
              const targetBps = (cfg.bandwidthKbps * 1000) / 8
              const expectedTimeMs = (dataLen / targetBps) * 1000
              if (expectedTimeMs > elapsedMs) {
                artificialDelayMs += expectedTimeMs - elapsedMs
              }
            }

            if (artificialDelayMs > 0) {
              setTimeout(() => {
                const totalElapsedSec = (performance.now() - startTime) / 1000
                const speedKbps = totalElapsedSec > 0 ? (dataLen * 8) / (totalElapsedSec * 1000) : 0
                setDownloadSpeedKbps(Math.round(speedKbps))
                originalOnSuccess(response, stats, ctx, networkDetails)
              }, artificialDelayMs)
            } else {
              const totalElapsedSec = elapsedMs / 1000
              const speedKbps = totalElapsedSec > 0 ? (dataLen * 8) / (totalElapsedSec * 1000) : 0
              setDownloadSpeedKbps(Math.round(speedKbps))
              originalOnSuccess(response, stats, ctx, networkDetails)
            }
          }

          super.load(context, config, callbacks)
        }
      }

      const hls = new Hls({
        loader: ThrottledLoader as any,
        enableWorker: true,
        lowLatencyMode: true,
        backBufferLength: 30,
      })

      hls.loadSource(url)
      hls.attachMedia(video)

      hls.on(Hls.Events.MANIFEST_PARSED, (_, data) => {
        addLog(`Manifest parsed: ${data.levels.length} variants detected`, 'success')
      })

      hls.on(Hls.Events.LEVEL_SWITCHED, (_, data) => {
        const level = hls.levels[data.level]
        if (level) {
          const res = `${level.width}x${level.height}`
          const bitrate = level.bitrate
          setActiveLevel(res)
          setActiveBitrate(bitrate)
          addLog(`ABR Switched to level #${data.level}: ${res} @ ${(bitrate / 1000000).toFixed(2)} Mbps`, 'warning')
        }
      })

      hls.on(Hls.Events.ERROR, (_, data) => {
        if (data.fatal) {
          addLog(`Fatal streaming error: ${data.details}`, 'error')
          switch (data.type) {
            case Hls.ErrorTypes.NETWORK_ERROR:
              hls.startLoad()
              break
            case Hls.ErrorTypes.MEDIA_ERROR:
              hls.recoverMediaError()
              break
            default:
              hls.destroy()
              break
          }
        }
      })

      hlsRef.current = hls
    } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
      video.src = url
    }
  }

  useEffect(() => {
    loadStream(streamUrl)
    return () => {
      if (hlsRef.current) {
        hlsRef.current.destroy()
      }
    }
  }, [])

  // Video element event listeners
  useEffect(() => {
    const video = videoRef.current
    if (!video) return

    const handlePlay = () => setIsPlaying(true)
    const handlePause = () => setIsPlaying(false)

    const handleTimeUpdate = () => {
      setCurrentTime(video.currentTime)
      setDuration(video.duration || 0)

      // Compute forward buffer length in seconds
      const b = video.buffered
      let forward = 0
      for (let i = 0; i < b.length; i++) {
        if (video.currentTime >= b.start(i) && video.currentTime <= b.end(i)) {
          forward = b.end(i) - video.currentTime
          break
        }
      }
      setForwardBuffer(forward)

      // Quality dropped frames
      if (typeof (video as any).getVideoPlaybackQuality === 'function') {
        const q = (video as any).getVideoPlaybackQuality()
        setDroppedFrames(q.droppedVideoFrames)
      }
    }

    const handleWaiting = () => {
      stallStartRef.current = performance.now()
      setStallCount((c) => c + 1)
      addLog('Player rebuffering (forward buffer underrun)...', 'error')
    }

    const handlePlaying = () => {
      if (stallStartRef.current) {
        const stallMs = performance.now() - stallStartRef.current
        setStallDurationSec((d) => d + stallMs / 1000)
        stallStartRef.current = null
        addLog(`Playback resumed after ${(stallMs / 1000).toFixed(2)}s stall`, 'success')
      }
    }

    video.addEventListener('play', handlePlay)
    video.addEventListener('pause', handlePause)
    video.addEventListener('timeupdate', handleTimeUpdate)
    video.addEventListener('waiting', handleWaiting)
    video.addEventListener('playing', handlePlaying)

    return () => {
      video.removeEventListener('play', handlePlay)
      video.removeEventListener('pause', handlePause)
      video.removeEventListener('timeupdate', handleTimeUpdate)
      video.removeEventListener('waiting', handleWaiting)
      video.removeEventListener('playing', handlePlaying)
    }
  }, [])

  // Draw Forward Buffer Timeline Canvas
  useEffect(() => {
    const canvas = canvasRef.current
    const video = videoRef.current
    if (!canvas || !video || !duration) return

    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const width = canvas.width
    const height = canvas.height

    // Clear
    ctx.fillStyle = '#090d16'
    ctx.fillRect(0, 0, width, height)

    // Draw buffered ranges
    const b = video.buffered
    ctx.fillStyle = 'rgba(56, 189, 248, 0.4)'
    for (let i = 0; i < b.length; i++) {
      const startX = (b.start(i) / duration) * width
      const endX = (b.end(i) / duration) * width
      ctx.fillRect(startX, 4, Math.max(2, endX - startX), height - 8)
    }

    // Draw played range
    const currentX = (currentTime / duration) * width
    ctx.fillStyle = 'rgba(99, 102, 241, 0.6)'
    ctx.fillRect(0, 4, currentX, height - 8)

    // Draw playhead cursor
    ctx.fillStyle = '#ffffff'
    ctx.fillRect(currentX - 1, 0, 3, height)
  }, [currentTime, duration, forwardBuffer])

  const togglePlay = () => {
    const video = videoRef.current
    if (!video) return
    if (video.paused) {
      video.play().catch(() => {})
    } else {
      video.pause()
    }
  }

  const restartVideo = () => {
    const video = videoRef.current
    if (!video) return
    video.currentTime = 0
    video.play().catch(() => {})
  }

  return (
    <div className="space-y-6">
      {/* Stream URL Bar */}
      <div className="bg-slate-900 border border-slate-800 rounded-2xl p-4 flex flex-col sm:flex-row items-center gap-3">
        <div className="flex items-center space-x-2 text-slate-400 text-xs shrink-0">
          <Film className="w-4 h-4 text-sky-400" />
          <span className="font-semibold text-slate-300">Stream URL:</span>
        </div>
        <input
          type="text"
          value={streamUrl}
          onChange={(e) => setStreamUrl(e.target.value)}
          placeholder="https://.../master.m3u8"
          className="flex-1 w-full bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2 text-xs font-mono text-slate-200 focus:outline-none focus:border-sky-500"
        />
        <button
          onClick={() => loadStream(streamUrl)}
          className="w-full sm:w-auto px-4 py-2 rounded-xl text-xs font-semibold bg-sky-500 hover:bg-sky-400 text-white transition-colors cursor-pointer shrink-0"
        >
          Load Stream
        </button>
      </div>

      {/* Network Bandwidth Throttle Simulator */}
      <ThrottleControl
        currentConfig={throttleConfig}
        onChangeConfig={(cfg) => {
          setThrottleConfig(cfg)
          addLog(`Network throttler updated: ${cfg.label}`, 'warning')
        }}
      />

      {/* Main Video View & Telemetry Gauges */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Video Canvas & Controls */}
        <div className="lg:col-span-7 bg-slate-900 border border-slate-800 rounded-2xl p-4 flex flex-col">
          <div className="relative rounded-xl overflow-hidden bg-black aspect-video flex items-center justify-center">
            <video
              ref={videoRef}
              controls={false}
              playsInline
              className="w-full h-full object-contain"
            />

            {/* Custom overlay controls */}
            <div className="absolute bottom-3 left-3 right-3 flex items-center justify-between bg-slate-950/80 backdrop-blur px-3 py-2 rounded-xl border border-slate-800 text-xs">
              <div className="flex items-center space-x-2">
                <button
                  onClick={togglePlay}
                  className="p-1.5 rounded-lg bg-sky-500 hover:bg-sky-400 text-white transition-colors"
                >
                  {isPlaying ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
                </button>
                <button
                  onClick={restartVideo}
                  className="p-1.5 rounded-lg hover:bg-slate-800 text-slate-300 transition-colors"
                >
                  <RotateCcw className="w-4 h-4" />
                </button>
                <span className="font-mono text-slate-300 text-[11px] ml-2">
                  {currentTime.toFixed(1)}s / {duration.toFixed(1)}s
                </span>
              </div>

              <div className="flex items-center space-x-2">
                <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-sky-500/20 text-sky-300 border border-sky-500/30">
                  {activeLevel}
                </span>
                {activeBitrate > 0 && (
                  <span className="text-[11px] font-mono text-slate-400">
                    {(activeBitrate / 1000000).toFixed(2)} Mbps
                  </span>
                )}
              </div>
            </div>
          </div>

          {/* Forward Buffer Visual Timeline */}
          <div className="mt-4 space-y-1.5">
            <div className="flex justify-between text-xs">
              <span className="text-slate-400 font-medium">Forward Buffer Timeline:</span>
              <span className="font-mono text-sky-400 font-bold">
                {forwardBuffer.toFixed(2)}s buffered ahead
              </span>
            </div>
            <canvas
              ref={canvasRef}
              width={600}
              height={24}
              className="w-full h-6 rounded-lg border border-slate-800"
            />
          </div>
        </div>

        {/* Real-time Telemetry Dashboard */}
        <div className="lg:col-span-5 bg-slate-900 border border-slate-800 rounded-2xl p-5 flex flex-col justify-between space-y-4">
          <div>
            <div className="flex items-center justify-between pb-3 border-b border-slate-800 mb-3">
              <div className="flex items-center space-x-2">
                <Activity className="w-4 h-4 text-sky-400" />
                <h4 className="text-sm font-semibold text-white">Live Telemetry</h4>
              </div>
              <span className="flex items-center space-x-1.5 text-[10px] text-emerald-400">
                <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                <span>Monitoring</span>
              </span>
            </div>

            {/* Metrics Grid */}
            <div className="grid grid-cols-2 gap-3 text-xs">
              <div className="bg-slate-950 p-3 rounded-xl border border-slate-800">
                <div className="flex items-center space-x-1.5 text-slate-400 text-[11px]">
                  <Clock className="w-3.5 h-3.5 text-sky-400" />
                  <span>Forward Buffer</span>
                </div>
                <div
                  className={`text-xl font-bold font-mono mt-1 ${
                    forwardBuffer < 2.0
                      ? 'text-rose-400'
                      : forwardBuffer < 5.0
                      ? 'text-amber-400'
                      : 'text-emerald-400'
                  }`}
                >
                  {forwardBuffer.toFixed(2)}s
                </div>
                <div className="text-[10px] text-slate-500 mt-0.5">
                  {forwardBuffer < 2.0 ? 'High stall risk' : 'Healthy buffer'}
                </div>
              </div>

              <div className="bg-slate-950 p-3 rounded-xl border border-slate-800">
                <div className="flex items-center space-x-1.5 text-slate-400 text-[11px]">
                  <Gauge className="w-3.5 h-3.5 text-indigo-400" />
                  <span>Throughput</span>
                </div>
                <div className="text-xl font-bold font-mono text-white mt-1">
                  {downloadSpeedKbps > 1000
                    ? `${(downloadSpeedKbps / 1000).toFixed(1)} Mbps`
                    : `${downloadSpeedKbps} Kbps`}
                </div>
                <div className="text-[10px] text-slate-500 mt-0.5">Chunk download speed</div>
              </div>

              <div className="bg-slate-950 p-3 rounded-xl border border-slate-800">
                <div className="flex items-center space-x-1.5 text-slate-400 text-[11px]">
                  <AlertTriangle className="w-3.5 h-3.5 text-amber-400" />
                  <span>Buffer Stalls</span>
                </div>
                <div
                  className={`text-xl font-bold font-mono mt-1 ${
                    stallCount > 0 ? 'text-rose-400' : 'text-slate-200'
                  }`}
                >
                  {stallCount}
                </div>
                <div className="text-[10px] text-slate-500 mt-0.5">
                  Duration: {stallDurationSec.toFixed(1)}s
                </div>
              </div>

              <div className="bg-slate-950 p-3 rounded-xl border border-slate-800">
                <div className="flex items-center space-x-1.5 text-slate-400 text-[11px]">
                  <Film className="w-3.5 h-3.5 text-purple-400" />
                  <span>Dropped Frames</span>
                </div>
                <div className="text-xl font-bold font-mono text-white mt-1">{droppedFrames}</div>
                <div className="text-[10px] text-slate-500 mt-0.5">Hardware decode drops</div>
              </div>
            </div>
          </div>

          {/* Event Logs Stream */}
          <div>
            <div className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-2">
              Streaming Telemetry Log
            </div>
            <div className="h-36 overflow-y-auto bg-slate-950 rounded-xl p-3 border border-slate-800 text-[11px] font-mono space-y-1.5">
              {eventLogs.map((log, i) => (
                <div key={i} className="flex items-start space-x-2">
                  <span className="text-slate-600 select-none">{log.time}</span>
                  <span
                    className={
                      log.type === 'error'
                        ? 'text-rose-400'
                        : log.type === 'warning'
                        ? 'text-amber-400'
                        : log.type === 'success'
                        ? 'text-emerald-400'
                        : 'text-slate-300'
                    }
                  >
                    {log.text}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
