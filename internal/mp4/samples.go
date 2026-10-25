package mp4

import (
	"bytes"
	"encoding/binary"
)

// Helper to write a 4-byte string
func writeFourCC(buf *bytes.Buffer, fourcc string) {
	buf.WriteString(fourcc[:4])
}

// Helper to create a box with length prefix
func makeBox(boxType string, payload []byte) []byte {
	size := uint32(8 + len(payload))
	var buf bytes.Buffer
	var sizeBuf [4]byte
	binary.BigEndian.PutUint32(sizeBuf[:], size)
	buf.Write(sizeBuf[:])
	writeFourCC(&buf, boxType)
	buf.Write(payload)
	return buf.Bytes()
}

// GenerateSampleMP4 builds a valid binary ISO MP4 container.
// If fastStart is true, moov is placed before mdat; if false, mdat precedes moov.
func GenerateSampleMP4(fastStart bool) []byte {
	// 1. Build ftyp box
	var ftypPayload bytes.Buffer
	writeFourCC(&ftypPayload, "isom")               // Major brand
	binary.Write(&ftypPayload, binary.BigEndian, uint32(512)) // Minor version
	writeFourCC(&ftypPayload, "isom")               // Compatible brands
	writeFourCC(&ftypPayload, "iso2")
	writeFourCC(&ftypPayload, "avc1")
	writeFourCC(&ftypPayload, "mp41")
	ftypBox := makeBox("ftyp", ftypPayload.Bytes())

	// 2. Build dummy mdat box (10 KB of simulated video NAL data)
	mdatData := make([]byte, 10240)
	for i := range mdatData {
		mdatData[i] = byte(i % 256)
	}
	mdatBox := makeBox("mdat", mdatData)

	// 3. Build sample table (stbl) inside trak
	// stsd (Sample Description: avc1)
	var avc1Payload bytes.Buffer
	avc1Payload.Write(make([]byte, 6)) // Reserved 6 bytes
	binary.Write(&avc1Payload, binary.BigEndian, uint16(1)) // Data reference index
	binary.Write(&avc1Payload, binary.BigEndian, uint16(0)) // Pre-defined
	binary.Write(&avc1Payload, binary.BigEndian, uint16(0)) // Reserved
	binary.Write(&avc1Payload, binary.BigEndian, [3]uint32{0, 0, 0}) // Pre-defined
	binary.Write(&avc1Payload, binary.BigEndian, uint16(1920)) // Width = 1920
	binary.Write(&avc1Payload, binary.BigEndian, uint16(1080)) // Height = 1080
	binary.Write(&avc1Payload, binary.BigEndian, uint32(0x00480000)) // Horiz resolution 72 dpi
	binary.Write(&avc1Payload, binary.BigEndian, uint32(0x00480000)) // Vert resolution 72 dpi
	binary.Write(&avc1Payload, binary.BigEndian, uint32(0)) // Data size
	binary.Write(&avc1Payload, binary.BigEndian, uint16(1)) // Frame count = 1
	compressorName := make([]byte, 32)
	copy(compressorName, "\x0aAVC Coding")
	avc1Payload.Write(compressorName)
	binary.Write(&avc1Payload, binary.BigEndian, uint16(0x0018)) // Depth 24
	binary.Write(&avc1Payload, binary.BigEndian, int16(-1))      // Pre-defined
	avc1Box := makeBox("avc1", avc1Payload.Bytes())

	var stsdPayload bytes.Buffer
	stsdPayload.Write([]byte{0, 0, 0, 0}) // Version (0) + Flags
	binary.Write(&stsdPayload, binary.BigEndian, uint32(1)) // Entry count = 1
	stsdPayload.Write(avc1Box)
	stsdBox := makeBox("stsd", stsdPayload.Bytes())

	// stts (Time-to-sample: 300 frames of 1000 ticks)
	var sttsPayload bytes.Buffer
	sttsPayload.Write([]byte{0, 0, 0, 0})
	binary.Write(&sttsPayload, binary.BigEndian, uint32(1)) // Entry count
	binary.Write(&sttsPayload, binary.BigEndian, uint32(300)) // Sample count
	binary.Write(&sttsPayload, binary.BigEndian, uint32(1000)) // Sample delta
	sttsBox := makeBox("stts", sttsPayload.Bytes())

	// stss (Sync samples / Keyframes: frame 1, 61, 121, 181, 241)
	var stssPayload bytes.Buffer
	stssPayload.Write([]byte{0, 0, 0, 0})
	keyframes := []uint32{1, 61, 121, 181, 241}
	binary.Write(&stssPayload, binary.BigEndian, uint32(len(keyframes)))
	for _, kf := range keyframes {
		binary.Write(&stssPayload, binary.BigEndian, kf)
	}
	stssBox := makeBox("stss", stssPayload.Bytes())

	// stsc (Sample-to-chunk: 10 chunks of 30 samples each)
	var stscPayload bytes.Buffer
	stscPayload.Write([]byte{0, 0, 0, 0})
	binary.Write(&stscPayload, binary.BigEndian, uint32(1)) // 1 entry
	binary.Write(&stscPayload, binary.BigEndian, uint32(1)) // First chunk
	binary.Write(&stscPayload, binary.BigEndian, uint32(30)) // Samples per chunk
	binary.Write(&stscPayload, binary.BigEndian, uint32(1)) // Sample description index
	stscBox := makeBox("stsc", stscPayload.Bytes())

	// stsz (Sample sizes: uniform 34 bytes per sample)
	var stszPayload bytes.Buffer
	stszPayload.Write([]byte{0, 0, 0, 0})
	binary.Write(&stszPayload, binary.BigEndian, uint32(34)) // Uniform size
	binary.Write(&stszPayload, binary.BigEndian, uint32(300)) // Sample count
	stszBox := makeBox("stsz", stszPayload.Bytes())

	// Calculate initial chunk offsets
	// 10 chunks inside mdat
	chunkCount := 10
	chunkSize := 1024

	// Build moov skeleton first without stco to measure size
	buildMoov := func(stcoOffsets []uint32) []byte {
		var stcoPayload bytes.Buffer
		stcoPayload.Write([]byte{0, 0, 0, 0}) // Version + Flags
		binary.Write(&stcoPayload, binary.BigEndian, uint32(len(stcoOffsets)))
		for _, off := range stcoOffsets {
			binary.Write(&stcoPayload, binary.BigEndian, off)
		}
		stcoBox := makeBox("stco", stcoPayload.Bytes())

		// Assemble stbl
		var stblPayload bytes.Buffer
		stblPayload.Write(stsdBox)
		stblPayload.Write(sttsBox)
		stblPayload.Write(stssBox)
		stblPayload.Write(stscBox)
		stblPayload.Write(stszBox)
		stblPayload.Write(stcoBox)
		stblBox := makeBox("stbl", stblPayload.Bytes())

		// vmhd
		var vmhdPayload bytes.Buffer
		vmhdPayload.Write([]byte{0, 0, 0, 1}) // Version + Flags
		binary.Write(&vmhdPayload, binary.BigEndian, uint16(0)) // Graphics mode
		binary.Write(&vmhdPayload, binary.BigEndian, [3]uint16{0, 0, 0}) // Opcolor
		vmhdBox := makeBox("vmhd", vmhdPayload.Bytes())

		// dinf
		drefBox := makeBox("dref", []byte{0, 0, 0, 0, 0, 0, 0, 0})
		dinfBox := makeBox("dinf", drefBox)

		// minf
		var minfPayload bytes.Buffer
		minfPayload.Write(vmhdBox)
		minfPayload.Write(dinfBox)
		minfPayload.Write(stblBox)
		minfBox := makeBox("minf", minfPayload.Bytes())

		// hdlr
		var hdlrPayload bytes.Buffer
		hdlrPayload.Write([]byte{0, 0, 0, 0}) // Version + Flags
		binary.Write(&hdlrPayload, binary.BigEndian, uint32(0)) // Predefined
		writeFourCC(&hdlrPayload, "vide") // Handler type
		binary.Write(&hdlrPayload, binary.BigEndian, [3]uint32{0, 0, 0}) // Reserved
		hdlrPayload.WriteString("VideoHandler\x00")
		hdlrBox := makeBox("hdlr", hdlrPayload.Bytes())

		// mdhd
		var mdhdPayload bytes.Buffer
		mdhdPayload.Write([]byte{0, 0, 0, 0}) // Version 0
		binary.Write(&mdhdPayload, binary.BigEndian, uint32(0)) // Creation time
		binary.Write(&mdhdPayload, binary.BigEndian, uint32(0)) // Modification time
		binary.Write(&mdhdPayload, binary.BigEndian, uint32(30000)) // Timescale 30kHz
		binary.Write(&mdhdPayload, binary.BigEndian, uint32(300000)) // Duration (10s)
		binary.Write(&mdhdPayload, binary.BigEndian, uint16(0x55C4)) // Language "und"
		binary.Write(&mdhdPayload, binary.BigEndian, uint16(0)) // Predefined
		mdhdBox := makeBox("mdhd", mdhdPayload.Bytes())

		// mdia
		var mdiaPayload bytes.Buffer
		mdiaPayload.Write(mdhdBox)
		mdiaPayload.Write(hdlrBox)
		mdiaPayload.Write(minfBox)
		mdiaBox := makeBox("mdia", mdiaPayload.Bytes())

		// tkhd
		var tkhdPayload bytes.Buffer
		tkhdPayload.Write([]byte{0, 0, 0, 15}) // Version 0 + Track enabled/in movie flags
		binary.Write(&tkhdPayload, binary.BigEndian, uint32(0)) // Creation time
		binary.Write(&tkhdPayload, binary.BigEndian, uint32(0)) // Mod time
		binary.Write(&tkhdPayload, binary.BigEndian, uint32(1)) // Track ID = 1
		binary.Write(&tkhdPayload, binary.BigEndian, uint32(0)) // Reserved
		binary.Write(&tkhdPayload, binary.BigEndian, uint32(10000)) // Duration (10s at timescale 1000)
		tkhdPayload.Write(make([]byte, 8)) // Reserved
		binary.Write(&tkhdPayload, binary.BigEndian, uint16(0)) // Layer
		binary.Write(&tkhdPayload, binary.BigEndian, uint16(0)) // Alt group
		binary.Write(&tkhdPayload, binary.BigEndian, uint16(0)) // Volume
		binary.Write(&tkhdPayload, binary.BigEndian, uint16(0)) // Reserved
		// Matrix structure (unity matrix)
		matrix := [9]uint32{0x00010000, 0, 0, 0, 0x00010000, 0, 0, 0, 0x40000000}
		for _, m := range matrix {
			binary.Write(&tkhdPayload, binary.BigEndian, m)
		}
		binary.Write(&tkhdPayload, binary.BigEndian, uint32(1920<<16)) // Width 1920.0
		binary.Write(&tkhdPayload, binary.BigEndian, uint32(1080<<16)) // Height 1080.0
		tkhdBox := makeBox("tkhd", tkhdPayload.Bytes())

		// trak
		var trakPayload bytes.Buffer
		trakPayload.Write(tkhdBox)
		trakPayload.Write(mdiaBox)
		trakBox := makeBox("trak", trakPayload.Bytes())

		// mvhd
		var mvhdPayload bytes.Buffer
		mvhdPayload.Write([]byte{0, 0, 0, 0}) // Version 0
		binary.Write(&mvhdPayload, binary.BigEndian, uint32(0)) // Creation time
		binary.Write(&mvhdPayload, binary.BigEndian, uint32(0)) // Mod time
		binary.Write(&mvhdPayload, binary.BigEndian, uint32(1000)) // Timescale 1000
		binary.Write(&mvhdPayload, binary.BigEndian, uint32(10000)) // Duration 10s
		binary.Write(&mvhdPayload, binary.BigEndian, uint32(0x00010000)) // Rate 1.0
		binary.Write(&mvhdPayload, binary.BigEndian, uint16(0x0100)) // Volume 1.0
		mvhdPayload.Write(make([]byte, 10)) // Reserved
		for _, m := range matrix {
			binary.Write(&mvhdPayload, binary.BigEndian, m)
		}
		mvhdPayload.Write(make([]byte, 24)) // Predefined
		binary.Write(&mvhdPayload, binary.BigEndian, uint32(2)) // Next track ID = 2
		mvhdBox := makeBox("mvhd", mvhdPayload.Bytes())

		// moov
		var moovPayload bytes.Buffer
		moovPayload.Write(mvhdBox)
		moovPayload.Write(trakBox)
		return makeBox("moov", moovPayload.Bytes())
	}

	// Compute base chunk offsets
	initialOffsets := make([]uint32, chunkCount)
	dummyMoov := buildMoov(initialOffsets)
	moovSize := len(dummyMoov)

	var result bytes.Buffer

	if fastStart {
		// Layout: ftyp -> moov -> mdat
		// mdat payload starts at len(ftyp) + moovSize + 8 (mdat header)
		mdatPayloadStart := len(ftypBox) + moovSize + 8
		for i := 0; i < chunkCount; i++ {
			initialOffsets[i] = uint32(mdatPayloadStart + i*chunkSize)
		}
		finalMoov := buildMoov(initialOffsets)

		result.Write(ftypBox)
		result.Write(finalMoov)
		result.Write(mdatBox)
	} else {
		// Layout: ftyp -> mdat -> moov
		// mdat payload starts at len(ftyp) + 8 (mdat header)
		mdatPayloadStart := len(ftypBox) + 8
		for i := 0; i < chunkCount; i++ {
			initialOffsets[i] = uint32(mdatPayloadStart + i*chunkSize)
		}
		finalMoov := buildMoov(initialOffsets)

		result.Write(ftypBox)
		result.Write(mdatBox)
		result.Write(finalMoov)
	}

	return result.Bytes()
}
