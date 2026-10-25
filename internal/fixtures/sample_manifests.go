package fixtures

// ValidHLSMaster is a compliant multi-variant master playlist.
const ValidHLSMaster = `#EXTM3U
#EXT-X-VERSION:6
#EXT-X-INDEPENDENT-SEGMENTS

#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio-aac",NAME="English",DEFAULT=YES,AUTOSELECT=YES,LANGUAGE="en",URI="audio/en.m3u8"
#EXT-X-MEDIA:TYPE=SUBTITLES,GROUP-ID="subs",NAME="English",DEFAULT=NO,AUTOSELECT=YES,LANGUAGE="en",URI="subs/en.m3u8"

#EXT-X-STREAM-INF:BANDWIDTH=6000000,AVERAGE-BANDWIDTH=5500000,RESOLUTION=1920x1080,FRAME-RATE=60.000,CODECS="avc1.64002a,mp4a.40.2",AUDIO="audio-aac",SUBTITLES="subs"
1080p/index.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=3500000,AVERAGE-BANDWIDTH=3200000,RESOLUTION=1280x720,FRAME-RATE=60.000,CODECS="avc1.64001f,mp4a.40.2",AUDIO="audio-aac",SUBTITLES="subs"
720p/index.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=1800000,AVERAGE-BANDWIDTH=1600000,RESOLUTION=854x480,FRAME-RATE=30.000,CODECS="avc1.4d401f,mp4a.40.2",AUDIO="audio-aac",SUBTITLES="subs"
480p/index.m3u8

#EXT-X-STREAM-INF:BANDWIDTH=800000,AVERAGE-BANDWIDTH=750000,RESOLUTION=640x360,FRAME-RATE=30.000,CODECS="avc1.4d401e,mp4a.40.2",AUDIO="audio-aac",SUBTITLES="subs"
360p/index.m3u8
`

// ValidHLSMedia is a compliant VOD media playlist.
const ValidHLSMedia = `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:6
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-PLAYLIST-TYPE:VOD

#EXTINF:6.006,
segment_0.ts
#EXTINF:5.998,
segment_1.ts
#EXTINF:6.002,
segment_2.ts
#EXTINF:6.000,
segment_3.ts
#EXTINF:5.995,
segment_4.ts
#EXTINF:3.450,
segment_5.ts
#EXT-X-ENDLIST
`

// BrokenHLSMedia has duration drift, target duration overflow, missing ENDLIST.
const BrokenHLSMedia = `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:4
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-PLAYLIST-TYPE:VOD

#EXTINF:4.000,
segment_0.ts
#EXTINF:6.850,
segment_1.ts
#EXTINF:1.200,
segment_2.ts
#EXTINF:5.400,
segment_3.ts
#EXTINF:8.200,
segment_4.ts
`

// ValidDASHManifest is a compliant MPEG-DASH XML manifest.
const ValidDASHManifest = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011"
     profiles="urn:mpeg:dash:profile:isoff-live:2011"
     type="static"
     mediaPresentationDuration="PT1M0.0S"
     minBufferTime="PT2.0S">
  <Period id="1" duration="PT1M0.0S">
    <AdaptationSet id="0" contentType="video" mimeType="video/mp4" subsegmentAlignment="true">
      <Representation id="v-1080p" bandwidth="5000000" width="1920" height="1080" frameRate="30" codecs="avc1.640028">
        <SegmentTemplate timescale="1000" duration="4000" media="v-1080p-$Number$.m4s" initialization="v-1080p-init.mp4"/>
      </Representation>
      <Representation id="v-720p" bandwidth="2800000" width="1280" height="720" frameRate="30" codecs="avc1.4d401f">
        <SegmentTemplate timescale="1000" duration="4000" media="v-720p-$Number$.m4s" initialization="v-720p-init.mp4"/>
      </Representation>
      <Representation id="v-360p" bandwidth="900000" width="640" height="360" frameRate="30" codecs="avc1.4d401e">
        <SegmentTemplate timescale="1000" duration="4000" media="v-360p-$Number$.m4s" initialization="v-360p-init.mp4"/>
      </Representation>
    </AdaptationSet>
    <AdaptationSet id="1" contentType="audio" mimeType="audio/mp4" codecs="mp4a.40.2">
      <Representation id="a-128k" bandwidth="128000" audioSamplingRate="48000">
        <SegmentTemplate timescale="1000" duration="4000" media="a-128k-$Number$.m4s" initialization="a-128k-init.mp4"/>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>
`
