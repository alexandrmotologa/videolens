# VideoLens Architecture and Technical Specifications

This document describes the engineering design, binary parsing protocols, validation rules, and algorithms implemented in VideoLens.

---

## 1. ISO Base Media File Format (ISOBMFF) Box Parser

### 1.1 Box Header Binary Layout
An ISO box (or QuickTime atom) begins with an 8-byte header:
- **Bytes 0-3**: 32-bit unsigned big-endian integer representing box length in bytes.
- **Bytes 4-7**: 4-byte ASCII character string identifying box FourCC type (`ftyp`, `moov`, `mdat`, etc.).

When box length is 1 (`0x00000001`), the box uses 64-bit extended length (`largesize`), stored in bytes 8-15 as a 64-bit unsigned big-endian integer. When box length is 0, the box extends to the end of the file (common for trailing `mdat` boxes).

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                          Box Size                             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                          Box Type (FourCC)                    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|       Extended Size (64-bit, present only if Size == 1)       |
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### 1.2 FullBox Extension
FullBoxes extend the standard header with:
- **Version**: 1 byte (0 or 1).
- **Flags**: 3 bytes (24-bit bitmask).

Boxes implementing FullBox include `mvhd`, `tkhd`, `mdhd`, `hdlr`, `vmhd`, `smhd`, `stsd`, `stts`, `ctts`, `stss`, `stsz`, `stco`, and `co64`.

### 1.3 Box Hierarchy
VideoLens parses container boxes recursively:
```
moov (Movie Metadata)
 ├── mvhd (Movie Header: timescale, duration)
 ├── trak (Track Container)
 │    ├── tkhd (Track Header: track ID, dimensions)
 │    ├── edts (Edit Container)
 │    └── mdia (Media Information)
 │         ├── mdhd (Media Header: timescale, language)
 │         ├── hdlr (Handler: vide, soun, hint)
 │         └── minf (Media Information)
 │              ├── vmhd / smhd (Video / Sound Media Header)
 │              ├── dinf (Data Information: dref)
 │              └── stbl (Sample Table)
 │                   ├── stsd (Sample Description: avc1, hvc1, mp4a)
 │                   ├── stts (Time-to-Sample)
 │                   ├── ctts (Composition Time-to-Sample)
 │                   ├── stss (Sync Samples / Keyframes)
 │                   ├── stsc (Sample-to-Chunk)
 │                   ├── stsz (Sample Sizes)
 │                   └── stco / co64 (Chunk Offsets)
 └── mdat (Media Data: compressed video and audio frames)
```

---

## 2. Fast-Start Remediation Algorithm

### 2.1 Problem Definition
In standard video capture or transcode pipelines, media data frames are written sequentially to the file before metadata is finalized. As a result, the `moov` atom is appended to the tail of the file after the `mdat` atom.

When a web browser streams this file over HTTP Range requests, it cannot initialize decoding until it downloads the `moov` atom. For multi-gigabyte video files, this causes multi-second playback startup delays. Fast-Start requires `moov` to be placed immediately after `ftyp`.

### 2.2 Offset Relocation Mechanics
Moving `moov` ahead of `mdat` shifts all media chunks by the exact byte size of `moov` (plus any alignment padding). Because the sample table references media chunks by absolute file offsets in `stco` (32-bit offsets) or `co64` (64-bit offsets), all entries must be adjusted:

$$\text{offset}' = \text{offset} + \Delta$$

where $\Delta = \text{size}(\text{moov})$.

### 2.3 32-bit to 64-bit Promotion
If shifting causes any 32-bit offset in `stco` to exceed $2^{32}-1$ ($4,294,967,295$ bytes), the box must be converted to `co64`. This increases the size of `moov` itself by $4 \times N$ bytes (where $N$ is the number of chunk entries), requiring an iterative offset calculation until fixed-point convergence.

---

## 3. RFC 8216 HLS Validation Rules

VideoLens implements the following validation criteria for HTTP Live Streaming:

1. **Target Duration Bound (RFC 8216 §4.3.3.1)**:
   Every segment duration specified by `#EXTINF` must be less than or equal to `#EXT-X-TARGETDURATION`. Segments exceeding target duration by more than 0.5s are flagged as critical errors.
2. **Segment Duration Jitter / Drift**:
   Calculates duration standard deviation ($\sigma$). Variations exceeding 15% indicate unaligned GOP encoding across segment boundaries.
3. **Discontinuity Alignment (RFC 8216 §4.3.2.3)**:
   Any change in video format, encoding parameters, resolution, or timestamp resets must be preceded by `#EXT-X-DISCONTINUITY`.
4. **Sequence Number Monotonicity (RFC 8216 §4.3.3.2)**:
   Media sequence numbers (`#EXT-X-MEDIA-SEQUENCE`) must increment strictly monotonically.
5. **Codec Parameter Syntax (RFC 6381)**:
   Validates codec strings such as `avc1.4d401f`, `mp4a.40.2`, `hvc1.1.6.L93.B0`.

---

## 4. ABR Bitrate Ladder Engine

### 4.1 Bits Per Pixel (BPP)
Bits Per Pixel measures data rate relative to visual display throughput:

$$\text{BPP} = \frac{\text{Bitrate}}{\text{Width} \times \text{Height} \times \text{FrameRate}}$$

### 4.2 Benchmark Ranges
- **H.264 / AVC**:
  - Starved: $< 0.05$
  - Recommended: $0.07 - 0.15$
  - Saturated: $> 0.20$
- **HEVC / H.265 / VP9**:
  - Recommended: $0.04 - 0.09$
- **AV1**:
  - Recommended: $0.03 - 0.07$

### 4.3 Ladder Step Ratio
The step ratio between adjacent ladder variants $R_{i}$ and $R_{i+1}$ is:

$$\text{Ratio} = \frac{\text{Bitrate}_{i+1}}{\text{Bitrate}_{i}}$$

- **Optimal Range**: $1.50\times - 2.00\times$.
- **High Step Warning ($> 2.5\times$)**: Players on fluctuating connections drop playback quality severely or stall before upshifting.
- **Low Step Warning ($< 1.3\times$)**: Inefficient encoding rung, consuming transcode computing and storage without human-perceptible quality differentiation.
