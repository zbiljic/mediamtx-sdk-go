package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zbiljic/mediamtx-sdk-go"
)

func main() {
	baseURL := "http://localhost:9997"

	// Create a new MediaMTX client
	client, err := mediamtx.NewClient(baseURL)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	fmt.Println("MediaMTX SDK Go Example")
	fmt.Println("========================")

	// Example 1: Get global configuration
	fmt.Println()
	fmt.Println("1. Getting global configuration...")
	config, err := client.ConfigGlobalGet(ctx)
	if err != nil {
		log.Printf("Failed to get global config: %v", err)
	} else {
		switch config := config.(type) {
		case *mediamtx.GlobalConf:
			fmt.Printf("   - API enabled: %t\n", config.API.Value)
			fmt.Printf("   - Log level: %s\n", config.LogLevel.Value)
			fmt.Printf("   - RTSP enabled: %t\n", config.Rtsp.Value)
			fmt.Printf("   - HLS enabled: %t\n", config.Hls.Value)
			fmt.Printf("   - WebRTC enabled: %t\n", config.Webrtc.Value)
		default:
			log.Fatalf("Unexpected type: %T", config)
		}
	}

	// Example 2: List all active paths
	fmt.Println()
	fmt.Println("2. Listing active paths...")
	paths, err := client.PathsList(ctx, mediamtx.PathsListParams{
		Page:         mediamtx.NewOptInt(0),
		ItemsPerPage: mediamtx.NewOptInt(100),
	})
	if err != nil {
		log.Printf("Failed to list paths: %v", err)
	} else {
		switch paths := paths.(type) {
		case *mediamtx.PathList:
			fmt.Printf("   Found %d active paths:\n", paths.ItemCount.Value)
			for i, path := range paths.Items {
				fmt.Printf("   %d. %s (ready: %t)\n", i+1, path.Name.Value, path.Ready.Value)
				if path.Source.IsSet() {
					fmt.Printf("      Source: %s (ID: %s)\n", path.Source.Value.Type.Value, path.Source.Value.ID.Value)
				}
			}
		default:
			log.Fatalf("Unexpected type: %T", paths)
		}
	}

	// Example 3: Monitor RTSP sessions
	fmt.Println()
	fmt.Println("3. Monitoring RTSP sessions...")
	rtspSessions, err := client.RtspSessionsList(ctx, mediamtx.RtspSessionsListParams{
		Page:         mediamtx.NewOptInt(0),
		ItemsPerPage: mediamtx.NewOptInt(100),
	})
	if err != nil {
		log.Printf("Failed to list RTSP sessions: %v", err)
	} else {
		switch rtspSessions := rtspSessions.(type) {
		case *mediamtx.RTSPSessionList:
			fmt.Printf("   Found %d RTSP sessions:\n", rtspSessions.ItemCount.Value)
			for i, session := range rtspSessions.Items {
				fmt.Printf("   %d. Session %s:\n", i+1, session.ID.Value)
				fmt.Printf("      Path: %s\n", session.Path.Value)
				fmt.Printf("      State: %s\n", session.State.Value)
				fmt.Printf("      Remote Address: %s\n", session.RemoteAddr.Value)
				fmt.Printf("      Bytes Received: %d\n", session.BytesReceived.Value)
				fmt.Printf("      Bytes Sent: %d\n", session.BytesSent.Value)
			}
		default:
			log.Fatalf("Unexpected type: %T", rtspSessions)
		}
	}

	// Example 4: Monitor RTMP connections
	fmt.Println()
	fmt.Println("4. Monitoring RTMP connections...")
	rtmpConns, err := client.RtmpConnsList(ctx, mediamtx.RtmpConnsListParams{
		Page:         mediamtx.NewOptInt(0),
		ItemsPerPage: mediamtx.NewOptInt(100),
	})
	if err != nil {
		log.Printf("Failed to list RTMP connections: %v", err)
	} else {
		switch rtmpConns := rtmpConns.(type) {
		case *mediamtx.RTMPConnList:
			fmt.Printf("   Found %d RTMP connections:\n", rtmpConns.ItemCount.Value)
			for i, conn := range rtmpConns.Items {
				fmt.Printf("   %d. Connection %s:\n", i+1, conn.ID.Value)
				fmt.Printf("      Path: %s\n", conn.Path.Value)
				fmt.Printf("      State: %s\n", conn.State.Value)
				fmt.Printf("      Remote Address: %s\n", conn.RemoteAddr.Value)
			}
		default:
			log.Fatalf("Unexpected type: %T", rtmpConns)
		}
	}

	// Example 5: List HLS muxers
	fmt.Println()
	fmt.Println("5. Monitoring HLS muxers...")
	hlsMuxers, err := client.HlsMuxersList(ctx, mediamtx.HlsMuxersListParams{
		Page:         mediamtx.NewOptInt(0),
		ItemsPerPage: mediamtx.NewOptInt(100),
	})
	if err != nil {
		log.Printf("Failed to list HLS muxers: %v", err)
	} else {
		switch hlsMuxers := hlsMuxers.(type) {
		case *mediamtx.HLSMuxerList:
			fmt.Printf("   Found %d HLS muxers:\n", hlsMuxers.ItemCount.Value)
			for i, muxer := range hlsMuxers.Items {
				fmt.Printf("   %d. Muxer for path: %s\n", i+1, muxer.Path.Value)
				fmt.Printf("      Created: %s\n", muxer.Created.Value)
				fmt.Printf("      Bytes Sent: %d\n", muxer.BytesSent.Value)
			}
		default:
			log.Fatalf("Unexpected type: %T", hlsMuxers)
		}
	}

	// Example 6: List recordings
	fmt.Println()
	fmt.Println("6. Checking recordings...")
	recordings, err := client.RecordingsList(ctx, mediamtx.RecordingsListParams{
		Page:         mediamtx.NewOptInt(0),
		ItemsPerPage: mediamtx.NewOptInt(100),
	})
	if err != nil {
		log.Printf("Failed to list recordings: %v", err)
	} else {
		switch recordings := recordings.(type) {
		case *mediamtx.RecordingList:
			fmt.Printf("   Found %d recordings:\n", recordings.ItemCount.Value)
			for i, recording := range recordings.Items {
				fmt.Printf("   %d. Recording: %s (%d segments)\n",
					i+1, recording.Name.Value, len(recording.Segments))
			}
		default:
			log.Fatalf("Unexpected type: %T", recordings)
		}
	}

	// Example 7: Path configuration management
	fmt.Println()
	fmt.Println("7. Path configuration example...")
	testPathName := "sdk-test-path"

	// Check if path already exists
	_, err = client.ConfigPathsGet(ctx, mediamtx.ConfigPathsGetParams{
		Name: testPathName,
	})
	if err == nil {
		fmt.Printf("   Path '%s' already exists, deleting it first...\n", testPathName)
		if _, err := client.ConfigPathsDelete(ctx, mediamtx.ConfigPathsDeleteParams{
			Name: testPathName,
		}); err != nil {
			log.Printf("Failed to delete existing path: %v", err)
		}
	} else {
		fmt.Printf("   Path '%s' doesn't exist, creating new one...\n", testPathName)
	}

	// Create a new path configuration
	pathConfig := mediamtx.PathConf{
		Name:         mediamtx.NewOptString(testPathName),
		Source:       mediamtx.NewOptString("publisher"),
		Record:       mediamtx.NewOptBool(true),
		RecordPath:   mediamtx.NewOptString("./recordings/%path/%Y-%m-%d_%H-%M-%S-%f"),
		RecordFormat: mediamtx.NewOptString("fmp4"),
		MaxReaders:   mediamtx.NewOptInt(50),
	}

	// Add the path
	if _, err := client.ConfigPathsAdd(ctx, &pathConfig, mediamtx.ConfigPathsAddParams{
		Name: testPathName,
	}); err != nil {
		log.Printf("Failed to add path config: %v", err)
	} else {
		fmt.Printf("   ✓ Created path configuration for '%s'\n", testPathName)

		// Verify it was created
		createdPath, err := client.ConfigPathsGet(ctx, mediamtx.ConfigPathsGetParams{
			Name: testPathName,
		})
		if err != nil {
			log.Printf("Failed to verify path creation: %v", err)
		} else {
			switch createdPath := createdPath.(type) {
			case *mediamtx.PathConf:
				fmt.Printf("   ✓ Verified path creation - Record: %t, MaxReaders: %d\n",
					createdPath.Record.Value, createdPath.MaxReaders.Value)
			case *mediamtx.ConfigPathsGetNotFound:
				fmt.Printf("   ✗ Path '%s' not found\n", testPathName)
			default:
				log.Fatalf("Unexpected type: %T", createdPath)
			}
		}

		// Clean up - delete the test path
		if _, err := client.ConfigPathsDelete(ctx, mediamtx.ConfigPathsDeleteParams{
			Name: testPathName,
		}); err != nil {
			log.Printf("Failed to cleanup test path: %v", err)
		} else {
			fmt.Printf("   ✓ Cleaned up test path\n")
		}
	}

	fmt.Println()
	fmt.Println("✓ MediaMTX SDK Go example completed successfully")
}
