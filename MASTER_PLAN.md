# VideoLens: Master Implementation Plan

> Modern HLS and DASH Streaming Video Inspector, MP4 Atom Tree Profiler, and ABR Ladder Validator

VideoLens is an automated streaming media diagnostics workbench and binary container inspection engine. It lints HLS (.m3u8) playlists and MPEG-DASH (.mpd) manifests, parses binary ISO MP4 box trees, evaluates Fast-Start web streaming readiness with automatic offset remuxing, audits ABR bitrate ladders with Bits Per Pixel (BPP) calculations, and simulates adaptive player buffering under network throttling.

---

## 1. System Architecture

```
+-----------------------------------------------------------------------------------+
|                                     VIDEOLENS                                     |
+-----------------------------------------+-----------------------------------------+
                                          |
        +---------------------------------+---------------------------------+
        |                                                                   |
+-------v-------------------------+                 +-----------------------v-------+
|    HLS & DASH Manifest Linter   |                 |    Binary MP4 Atom Profiler   |
|  * RFC 8216 M3U8 Playlist Parser|                 |  * ISO/IEC 14496-12 Box Parser|
|  * MPEG-DASH MPD XML Validator  |                 |  * ftyp, moov, trak, stbl, mdat|
|  * Target Duration Drift Audit  |                 |  * Fast-Start Remediation     |
+-------+-------------------------+                 +-----------------------+-------+
        |                                                                   |
        +---------------------------------+---------------------------------+
                                          |
                        +-----------------v-----------------+
                        |    ABR Ladder & Codec Validator   |
                        |  * Resolution Progression Curves  |
                        |  * Bits Per Pixel (BPP) Quality   |
                        |  * Step Ratio & Inversion Audits  |
                        +-----------------+-----------------+
                                          |
                        +-----------------v-----------------+
                        |   Buffer & Throttle Simulator     |
                        |  * Network Bandwidth Throttling   |
                        |  * Forward Buffer Gauge (Seconds) |
                        |  * Rebuffering Stall Telemetry    |
                        +-----------------+-----------------+
                                          |
                        +-----------------v-----------------+
                        |   Interactive Web Studio & CLI    |
                        |  * Collapsible Atom Tree View     |
                        |  * Manifest Syntax Highlighter    |
                        |  * Live Hls.js Playback Monitor   |
                        +-----------------------------------+
```

---

## 2. Directory Structure

```
videolens/
├── cmd/
│   └── videolens/
│       └── main.go               # Unified CLI and HTTP server entrypoint
├── internal/
│   ├── hls/
│   │   ├── master_parser.go      # HLS master playlist parser (variants, audio, subtitles)
│   │   ├── media_parser.go       # HLS media playlist parser (#EXTINF, sequence, byte-range)
│   │   ├── validator.go          # RFC 8216 validation engine: duration drift, discontinuities
│   │   └── dash_parser.go        # MPEG-DASH XML manifest parser and validator
│   ├── mp4/
│   │   ├── reader.go             # Binary ISO box reader with 64-bit largesize support
│   │   ├── atom_tree.go          # Recursive box hierarchy constructor with codec metadata
│   │   ├── faststart.go          # Fast-Start detector and in-memory stco/co64 re-locator
│   │   └── samples.go            # Synthetic MP4 generator for instant local testing
│   ├── ladder/
│   │   ├── bitrate_curve.go      # Bits Per Pixel (BPP) calculator and step ratio auditor
│   │   └── recommendations.go    # Actionable bitrate ladder optimization engine
│   ├── server/
│   │   ├── api.go                # REST API handlers for linting, parsing, and remuxing
│   │   ├── proxy.go              # Media streaming proxy with CORS headers for test URLs
│   │   └── static.go             # Embedded React single-page application assets
│   ├── cli/
│   │   └── commands.go           # CLI command handlers (check, faststart, abr, serve)
│   └── fixtures/
│       └── sample_manifests.go   # Built-in valid and broken HLS/DASH test manifests
├── web/
│   ├── src/
│   │   ├── components/
│   │   │   ├── Navbar.tsx            # Top navigation bar with sample selector
│   │   │   ├── AtomTreeView.tsx      # Interactive collapsible MP4 box tree explorer
│   │   │   ├── ManifestLinter.tsx    # Syntax-highlighted manifest viewer with error badges
│   │   │   ├── AbrLadderChart.tsx    # Interactive Bitrate vs Resolution curve graph
│   │   │   ├── PlayerSandbox.tsx     # Hls.js player with buffer gauge and stall tracking
│   │   │   ├── ThrottleControl.tsx   # Network bandwidth limiter and latency injection
│   │   │   └── FastStartBanner.tsx   # Fast-Start status badge with 1-click remux download
│   │   ├── hooks/
│   │   │   └── useVideoInspector.ts  # State management for media inspection
│   │   ├── types/
│   │   │   └── media.ts              # TypeScript interfaces for API payloads
│   │   ├── App.tsx                   # Main tabbed workspace layout
│   │   ├── index.css                 # Tailwind CSS styles and custom utility classes
│   │   └── main.tsx                  # React 19 root mounting
│   ├── package.json
│   ├── vite.config.ts
│   ├── tailwind.config.js
│   └── tsconfig.json
├── docs/
│   ├── images/
│   │   └── videolens_demo.gif    # Animated walkthrough demonstration
│   └── architecture.md           # Engineering specifications and RFC compliance notes
├── go.mod
├── go.sum
├── Makefile
├── LICENSE
└── README.md
```

---

## 3. Detailed Component Specifications

### 3.1 Binary MP4 Box and Atom Profiler
- **Binary Reader (`internal/mp4/reader.go`)**:
  - Reads 32-bit big-endian length and 4-byte ASCII FourCC identifier.
  - Supports 64-bit extended length (`largesize`) when header length equals 1.
  - Supports unconstrained `mdat` extending to end of file when length equals 0.
  - Extracts full box version and 24-bit flags for ISO FullBox types.
- **Box Hierarchy (`internal/mp4/atom_tree.go`)**:
  - Recursive descent parser for container boxes: `moov`, `trak`, `edts`, `mdia`, `minf`, `dinf`, `stbl`, `mvex`, `moof`, `traf`.
  - Detailed metadata extraction:
    - `mvhd`: Timescale, duration, play rate, volume.
    - `tkhd`: Track ID, track duration, visual width and height.
    - `mdhd`: Track timescale, media duration, ISO 639-2 language code.
    - `hdlr`: Component type (`vide`, `soun`, `hint`, `meta`), component name.
    - `stsd`: Sample description table parsing codec FourCC (`avc1`, `hvc1`, `hev1`, `vp09`, `av01`, `mp4a`), visual resolution, color depth, audio channel count, sample rate.
    - `stts` and `ctts`: Time-to-sample and composition time tables.
    - `stss`: Sync sample table (keyframe / IDR index), calculating keyframe count and interval.
    - `stsz` / `stz2`: Sample size entries, computing total frame count and average frame size.
    - `stco` / `co64`: Chunk offset tables (32-bit and 64-bit).
    - `sidx`: Segment index for fragmented MP4 / DASH subsegment indexing.
- **Fast-Start Evaluation and Remediation (`internal/mp4/faststart.go`)**:
  - Validates whether `moov` occurs prior to `mdat`.
  - If `mdat` precedes `moov`:
    - Flags warning `NOT_FAST_START_OPTIMIZED` with byte distance and percentage into file.
    - Calculates estimated initial buffering latency based on connection speed.
  - Remediation engine (`RelocateMoov`):
    - Reads the MP4 file structure.
    - Extracts `moov` box into memory.
    - Adjusts all chunk offsets in `stco` and `co64` by adding the length of `moov`.
    - If any 32-bit `stco` offset overflows $2^{32}-1$, automatically promotes the box to `co64`.
    - Writes `ftyp`, relocated `moov`, and trailing `mdat` to the output stream.
    - Provides instant remux download through REST API (`POST /api/mp4/remediate`) and CLI (`videolens faststart`).

### 3.2 HLS and DASH Manifest Linter
- **HLS Parser and Linter (`internal/hls/master_parser.go`, `media_parser.go`, `validator.go`)**:
  - RFC 8216 and RFC 8217 compliant parser.
  - Master playlist analysis:
    - Validates `#EXT-X-STREAM-INF` variants: `BANDWIDTH`, `AVERAGE-BANDWIDTH`, `RESOLUTION`, `FRAME-RATE`, `CODECS`.
    - Validates `#EXT-X-MEDIA` rendition groups: audio, video, subtitles, closed captions.
  - Media playlist analysis:
    - Parses `#EXT-X-TARGETDURATION`, `#EXT-X-MEDIA-SEQUENCE`, `#EXT-X-DISCONTINUITY-SEQUENCE`, `#EXTINF`, `#EXT-X-BYTERANGE`, `#EXT-X-DISCONTINUITY`, `#EXT-X-KEY`, `#EXT-X-MAP`, `#EXT-X-PROGRAM-DATE-TIME`, `#EXT-X-ENDLIST`.
  - Validation rules:
    - Rule 1: Segment duration must not exceed `#EXT-X-TARGETDURATION + 0.5s`.
    - Rule 2: Segment duration drift (computes mean, variance, standard deviation of durations).
    - Rule 3: Missing `#EXT-X-DISCONTINUITY` tags when timeline resets or codec parameters switch.
    - Rule 4: Sequence number continuity checks across segments.
    - Rule 5: Codec string format validation (RFC 6381 conformance).
    - Rule 6: Audio and subtitle rendition mapping sanity.
    - Rule 7: Target duration integer rounding validation.
- **MPEG-DASH Parser and Validator (`internal/hls/dash_parser.go`)**:
  - Parses XML manifest: `MPD`, `Period`, `AdaptationSet`, `Representation`, `SegmentTemplate`, `SegmentTimeline`, `SegmentList`, `BaseURL`.
  - Validation rules:
    - Rule 1: `minBufferTime` vs segment duration ratio.
    - Rule 2: Representation bandwidth ordering within adaptation sets.
    - Rule 3: Timeline gaps and continuity issues in `SegmentTimeline` (`t`, `d`, `r`).
    - Rule 4: Initialization segment template validity.

### 3.3 ABR Bitrate Ladder Validator
- **Bitrate Curve and BPP Engine (`internal/ladder/bitrate_curve.go`)**:
  - Extracts all resolution and bitrate tiers.
  - Computes Bits Per Pixel:
    $$\text{BPP} = \frac{\text{Bitrate}}{\text{Width} \times \text{Height} \times \text{FPS}}$$
  - Quality thresholds:
    - H.264: Optimal 0.07 to 0.15 BPP. Flag under 0.05 (starved/artifacts) and over 0.20 (wasted bandwidth).
    - HEVC/VP9: Optimal 0.04 to 0.09 BPP.
    - AV1: Optimal 0.03 to 0.07 BPP.
  - Step ratio checks:
    - Ratio between adjacent bitrate rungs (optimal 1.5x to 2.0x).
    - Flags step ratios > 2.5x (harsh jumps causing player hunting and rebuffering).
    - Flags step ratios < 1.3x (redundant encoding rungs wasting storage and CDN bandwidth).
  - Inverted ladder detection:
    - Flags cases where higher resolution has lower bitrate, or higher bitrate has lower resolution.
  - ABR Health Score (0 to 100):
    - Weighted composite score evaluating resolution progression, BPP consistency, step ratios, and codec allocation.

### 3.4 Interactive Video Player Sandbox and Network Throttle Simulator
- **Player Studio (`web/src/components/PlayerSandbox.tsx`)**:
  - Hls.js engine for live and VOD stream playback.
  - Telemetry collection:
    - Forward buffer duration ($t_{\text{buffer}} = \text{buffered.end} - \text{currentTime}$).
    - Instantaneous download speed (Kbps and Mbps).
    - Active bitrate tier and resolution.
    - Total rebuffering stall count and aggregate stall duration.
    - Dropped frame count via `HTMLVideoElement.getVideoPlaybackQuality()`.
- **Network Throttle Simulator (`web/src/components/ThrottleControl.tsx`)**:
  - Intercepts segment chunk requests via custom Hls.js loader.
  - Presets:
    - Direct connection (unlimited)
    - Fiber / High-Speed (25 Mbps)
    - 4G LTE Good (8 Mbps)
    - 4G LTE Degraded (3 Mbps)
    - 3G Fast (1.5 Mbps)
    - 3G Slow (500 Kbps)
    - 2G / Edge (150 Kbps)
    - Dynamic "Tunnel Drop" (10 Mbps -> 400 Kbps for 8s -> recovery)
  - Custom bandwidth slider (50 Kbps to 50 Mbps) and latency injection (0ms to 1000ms).

---

## 4. Implementation Subtasks

### Task 1: Repository Setup and Portfolio Synchronization
- 1.1 Initialize Git repository, configure origin remote, set description and 12 topics.
- 1.2 Synchronize Phase 1 portfolio entries in `mtlg-site` and `mtlglabs-space`.
- 1.3 Create project documentation, license (MIT), and humanized README.md.

### Task 2: Binary ISO MP4 Parser and Fast-Start Remuxer
- 2.1 Implement `internal/mp4/reader.go` with 32-bit and 64-bit box parsing.
- 2.2 Implement `internal/mp4/atom_tree.go` with recursive box hierarchy constructor and codec extractor.
- 2.3 Implement `internal/mp4/faststart.go` with Fast-Start detection and in-memory offset relocator (`RelocateMoov`).
- 2.4 Implement `internal/mp4/samples.go` generating synthetic test MP4 files (both Fast-Start optimized and unoptimized).
- 2.5 Write automated unit tests verifying atom tree parsing and offset rewriting.

### Task 3: HLS RFC 8216 and MPEG-DASH Manifest Linter
- 3.1 Implement `internal/hls/master_parser.go` for multi-variant master playlists.
- 3.2 Implement `internal/hls/media_parser.go` for media playlists, segments, and byte ranges.
- 3.3 Implement `internal/hls/validator.go` with RFC 8216 rules: duration drift, target duration excess, discontinuity tags.
- 3.4 Implement `internal/hls/dash_parser.go` parsing MPEG-DASH `.mpd` XML elements and validating representation ladders.
- 3.5 Write unit tests against valid and broken manifest fixtures.

### Task 4: ABR Bitrate Ladder Engine
- 4.1 Implement `internal/ladder/bitrate_curve.go` computing BPP, ladder ratios, and inverted rungs.
- 4.2 Implement `internal/ladder/recommendations.go` generating actionable recommendations and composite health scores.
- 4.3 Write comprehensive unit tests for ladder evaluation.

### Task 5: HTTP Server, Streaming Proxy, and CLI Commands
- 5.1 Implement `internal/server/api.go` exposing `/api/hls/lint`, `/api/dash/lint`, `/api/mp4/inspect`, `/api/mp4/remediate`, `/api/ladder/analyze`, and `/api/samples`.
- 5.2 Implement `internal/server/proxy.go` providing CORS-enabled proxying for external test streams.
- 5.3 Implement `internal/cli/commands.go` providing CLI commands: `check`, `faststart`, `abr`, and `serve`.
- 5.4 Wire unified CLI and server entrypoint in `cmd/videolens/main.go`.

### Task 6: Modern React 19 Frontend Web Studio
- 6.1 Scaffold Vite + React 19 + TypeScript + Tailwind CSS in `web/`.
- 6.2 Build Atom Tree View component with collapsible nodes, hex preview, and Fast-Start badge.
- 6.3 Build Manifest Linter component with syntax-highlighted lines and error pills.
- 6.4 Build ABR Ladder Chart component with interactive visual curves and BPP gauges.
- 6.5 Build Player Sandbox with Hls.js, buffer health gauge, stall telemetry, and network throttle simulator.
- 6.6 Embed compiled web assets into Go binary using `embed.FS`.

### Task 7: Verification, Asset Generation, and Launch
- 7.1 Run full end-to-end tests across CLI, API, and web interface.
- 7.2 Capture UI screenshots and compile animated demo GIF in `docs/images/videolens_demo.gif`.
- 7.3 Execute MTLG portfolio sync Phase 2 (LIVE status, screenshots migration, sync both sites).
- 7.4 Broadcast launch update.
