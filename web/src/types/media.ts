export type TabType = 'linter' | 'mp4' | 'abr' | 'player'

export interface BoxNode {
  type: string
  offset: number
  headerSize: number
  payloadSize: number
  totalSize: number
  children?: BoxNode[]
  details?: Record<string, any>
}

export interface GOPInterval {
  index: number
  startFrame: number
  durationFrames: number
  durationSeconds: number
  startSeconds: number
}

export interface GOPProfile {
  totalKeyframes: number
  totalFrames: number
  cadenceType: 'FIXED' | 'VARIABLE' | 'SINGLE_KEYFRAME'
  meanGOPDuration: number
  minGOPDuration: number
  maxGOPDuration: number
  isFixedGOP: boolean
  spikeCount: number
  intervals: GOPInterval[]
  recommendation: string
}

export interface TrackSummary {
  trackId: number
  type: 'video' | 'audio' | 'hint' | 'meta' | 'unknown'
  codec: string
  durationSeconds: number
  timescale: number
  width?: number
  height?: number
  sampleRate?: number
  channels?: number
  keyframeCount?: number
  totalFrames?: number
  avgKeyframeSec?: number
  gop?: GOPProfile
}

export interface FastStartStatus {
  isFastStart: boolean
  moovOffset: number
  mdatOffset: number
  distance: number
  moovSize: number
  mdatSize: number
  totalSize: number
  moovPositionPercent: number
  initialLatencyEstimateMs: number
  statusBadge: 'OPTIMIZED' | 'NOT_OPTIMIZED' | 'MISSING_METADATA'
  message: string
}

export interface MP4Profile {
  fileSize: number
  durationSeconds: number
  timescale: number
  majorBrand: string
  compatibleBrands: string[]
  fastStart?: FastStartStatus
  tracks: TrackSummary[]
  rootBoxes: BoxNode[]
}

export interface LintIssue {
  code: string
  severity: 'ERROR' | 'WARNING' | 'INFO'
  line: number
  message: string
  recommendation?: string
}

export interface DurationMetrics {
  minDuration: number
  maxDuration: number
  meanDuration: number
  variance: number
  stdDev: number
  driftPercent: number
}

export interface VariantStream {
  bandwidth: number
  averageBandwidth?: number
  resolution?: string
  width?: number
  height?: number
  frameRate?: number
  codecs?: string
  audio?: string
  subtitles?: string
  uri: string
  lineNumber: number
}

export interface MediaRendition {
  type: string
  groupId: string
  name: string
  language?: string
  default: boolean
  autoSelect: boolean
  uri?: string
  lineNumber: number
}

export interface PartialSegment {
  index: number
  duration: number
  uri: string
  independent: boolean
  byteRange?: string
  lineNumber: number
}

export interface Segment {
  index: number
  duration: number
  title?: string
  uri: string
  lineNumber: number
  byteRangeLength?: number
  byteRangeOffset?: number
  discontinuity: boolean
  programDateTime?: string
  keyMethod?: string
  keyUri?: string
  parts?: PartialSegment[]
}

export interface HLSLintResult {
  isMaster: boolean
  version: number
  targetDuration?: number
  mediaSequence?: number
  totalDuration?: number
  segmentCount: number
  discontinuityCount: number
  playlistType?: string
  hasEndList: boolean
  isLowLatency?: boolean
  partTarget?: number
  totalPartsCount?: number
  durationMetrics?: DurationMetrics
  variants?: VariantStream[]
  renditions?: MediaRendition[]
  segments?: Segment[]
  rawLines: string[]
  issues: LintIssue[]
  healthScore: number
  ladder?: LadderAnalysis
}

export interface FixAction {
  code: string
  description: string
}

export interface FixResult {
  originalContent: string
  sanitizedContent: string
  fixesApplied: FixAction[]
  originalScore: number
  sanitizedScore: number
}

export interface DeviceCompatibility {
  codecString: string
  profileName: string
  levelNumber: number
  ios: boolean
  android: boolean
  smartTv: boolean
  roku: boolean
  desktopWeb: boolean
  legacyMobile: boolean
  notes: string[]
}

export interface AlignmentSegment {
  index: number
  duration: number
  pts: number
}

export interface VariantTrack {
  id: string
  resolution?: string
  bitrate?: number
  bandwidth?: number
  segments: AlignmentSegment[]
}

export interface AlignmentDiscrepancy {
  segmentIndex: number
  variantA: string
  variantB: string
  ptsA: number
  ptsB: number
  driftMs: number
  severity: string
  message: string
}

export interface AlignmentMatrix {
  variantCount: number
  segmentCount: number
  aligned: boolean
  maxDriftMs: number
  score: number
  discrepancies: AlignmentDiscrepancy[]
  summary: string
}

export interface DASHRepresentation {
  id: string
  mimeType: string
  bandwidth: number
  width?: number
  height?: number
  frameRate?: number
  codecs?: string
  sampleRate?: string
}

export interface DASHAdaptationSet {
  id: string
  contentType: string
  mimeType: string
  subsegmentAlignment: boolean
  representations: DASHRepresentation[]
}

export interface DASHLintResult {
  profiles: string
  type: string
  minBufferTimeSec: number
  durationSec: number
  periodCount: number
  adaptationSets: DASHAdaptationSet[]
  totalVariantsCount: number
  issues: LintIssue[]
  healthScore: number
  rawLines: string[]
}

export interface LadderRung {
  id: string
  resolution: string
  width: number
  height: number
  fps: number
  bitrate: number
  bitrateKbps: number
  bitrateMbps: number
  codec: string
  codecFamily: string
  bitsPerPixel: number
  stepRatio: number
  bppStatus: 'OPTIMAL' | 'STARVED' | 'SATURATED'
  evaluation: string
  compatibility?: DeviceCompatibility
}

export interface LadderAnalysis {
  rungs: LadderRung[]
  rungCount: number
  minBitrate: number
  maxBitrate: number
  dynamicRange: number
  healthScore: number
  issues: {
    code: string
    severity: string
    rungId?: string
    message: string
    recommendation: string
  }[]
  recommendations: string[]
}

export interface SampleItem {
  id: string
  name: string
  type: string
  description: string
  content?: string
  url?: string
}
