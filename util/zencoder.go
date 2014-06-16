package util

import(
	"github.com/brandscreen/zencoder"
	"strings"
)

func BuildZencoderSettings(inputUrl, outputUrl, notificationListener, outfile string) *zencoder.EncodingSettings {
	out600 := outfile + "_hls_600.m3u8"
	out1200 := outfile + "_hls_1200.m3u8"
	outPlaylist := outfile + ".m3u8"
	
	not600 := notificationListener
	if !strings.HasSuffix(not600, "/") {
		not600 += "/"
	}
	not600 += out600
	
	not1200 := notificationListener
	if !strings.HasSuffix(not1200, "/") {
		not1200 += "/"
	}
	not1200 += out1200
	
	zcsettings := &zencoder.EncodingSettings{
		Input: inputUrl,
		Test: false,
		Outputs: []*zencoder.OutputSettings {
			&zencoder.OutputSettings{
				Label: "hls_600",
				Size: "640x360",
				VideoBitrate: 600,
				BaseUrl: outputUrl,
				Filename: out600,
				Type: "segmented",
				Format: "ts",
				Headers: map[string]string{
					"x-amz-acl": "public-read",
				},
				Notifications: []*zencoder.NotificationSettings {
					&zencoder.NotificationSettings {
						Url: not600,
					},
				},
			},
			&zencoder.OutputSettings{
				Label: "hls_1200",
				Size: "1280x720",
				VideoBitrate: 1200,
				BaseUrl: outputUrl,
				Filename: out1200,
				Type: "segmented",
				Format: "ts",
				Headers: map[string]string{
					"x-amz-acl": "public-read",
				},
				Notifications: []*zencoder.NotificationSettings {
					&zencoder.NotificationSettings {
						Url: not1200,
					},
				},
			},
			&zencoder.OutputSettings{
				BaseUrl: outputUrl,
				Filename: outPlaylist,
				Type: "playlist",
				Streams: []*zencoder.StreamSettings {
					&zencoder.StreamSettings{
						Bandwidth: 600,
						Path: out600,
					},
					&zencoder.StreamSettings{
						Bandwidth: 1200,
						Path: out1200,
					},
				},
				Headers: map[string]string{
					"x-amz-acl": "public-read",
				},
			},
		},
	}
	return zcsettings
}
