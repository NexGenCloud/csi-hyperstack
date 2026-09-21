package hyperstack

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubernetes-csi/csi-lib-utils/protosanitizer"
)

// TestGetVolume tests the GetVolume method with different volume IDs
func TestGetVolume(t *testing.T) {
	// Get API credentials from environment variables
	apiKey := ""
	apiServer := ""

	if apiKey == "" || apiServer == "" {
		t.Skip("Skipping test: HYPERSTACK_API_KEY or HYPERSTACK_API_SERVER environment variables not set")
	}

	// Create a new Hyperstack client
	client := NewHyperstackClient(apiKey, apiServer, "test")
	hs := &Hyperstack{
		Client: client,
	}

	// Test cases with different volume IDs
	testCases := []struct {
		name     string
		volumeID int
	}{
		{"valid_volume_id", 903},
	}

	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Printf("Testing GetVolume with volumeID: %d\n", tc.volumeID)

			result, err := hs.GetVolume(ctx, tc.volumeID)
			if err != nil {
				t.Errorf("Error for volumeID %v\n", err)
				return
			}
			fmt.Printf("Result: %+v\n", protosanitizer.StripSecrets(*result))

			if result == nil {
				fmt.Printf("Volume %d not found (expected)\n", tc.volumeID)
				return
			} else {
				// fmt.Printf("Id: %+v\n", *result.Id)
				// fmt.Printf("Name: %+v\n", *result.Name)
				// fmt.Printf("Size: %+v\n", *result.Size)
				// fmt.Printf("Status: %+v\n", *result.Status)
				// fmt.Printf("Environment: %+v\n", *result.Environment.Name)
				// fmt.Printf("CreatedAt: %+v\n", *result.CreatedAt)
				// fmt.Printf("UpdatedAt: %+v\n", *result.UpdatedAt)
				// fmt.Printf("VolumeType: %+v\n", *result.VolumeType)
				// fmt.Printf("Available: %+v\n", *result.Status)
				fmt.Printf("Attachment: %+v\n", protosanitizer.StripSecrets(*result.Attachments))
				return
			}

			// if result.Name != nil {
			// 	fmt.Printf("Volume name: %s\n", *result.Name)
			// }

			// if result.Size != nil {
			// 	fmt.Printf("Volume size: %d GB\n", *result.Size)
			// }
		})
	}
}

func TestAttachVolumeToNode(t *testing.T) {
	apiKey := "ca172084-30c8-419d-9148-b88822a992d6"
	apiServer := "https://staging-infrahub-api.internal.ngbackend.cloud/v1"

	client := NewHyperstackClient(apiKey, apiServer, "test")
	hs := &Hyperstack{
		Client: client,
	}

	ctx := context.Background()
	testCases := []struct {
		name      string
		vmID      int
		volumeID  int
		expectNil bool
	}{
		{"valid_volume_id", 268040, 886, false},
		// {"non_existent_id", 1, 99999, true},
		// {"empty_id", 1, 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Printf("Testing AttachVolumeToNode with volumeID: %d\n", tc.volumeID)

			result, err := hs.AttachVolumeToNode(ctx, tc.vmID, tc.volumeID)
			fmt.Printf("Result: %+v\n", result)
			if err != nil {
				t.Errorf("Error for volumeID %d: %v\n", tc.volumeID, err)
				return
			}
			fmt.Printf("Result: %+v\n", result)
		})
	}
}

func TestDetachVolumeFromNode(t *testing.T) {
	apiKey := "ca172084-30c8-419d-9148-b88822a992d6"
	apiServer := "https://staging-infrahub-api.internal.ngbackend.cloud/v1"

	// Create a new Hyperstack client
	client := NewHyperstackClient(apiKey, apiServer, "test")
	hs := &Hyperstack{
		Client: client,
	}

	ctx := context.Background()
	testCases := []struct {
		name      string
		vmID      int
		volumeID  int
		expectNil bool
	}{
		{"valid_volume_id", 268047, 903, false},
		// {"non_existent_id", 1, 99999, true},
		// {"empty_id", 1, 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Printf("Testing DetachVolumeFromNode with volumeID: %d\n", tc.volumeID)

			result, err := hs.DetachVolumeFromNode(ctx, tc.vmID, tc.volumeID)
			if err != nil {
				if !tc.expectNil {
					t.Errorf("Unexpected error for volumeID %d: %v\n", tc.volumeID, err)
				} else {
					fmt.Printf("Expected error for volumeID %d: %v\n", tc.volumeID, err)
				}
				return
			}
			if result == nil {
				if !tc.expectNil {
					t.Errorf("Unexpected nil result for volumeID %d\n", tc.volumeID)
				} else {
					fmt.Printf("Expected nil result for volumeID %d\n", tc.volumeID)
				}
				return
			}
			if tc.expectNil {
				t.Errorf("Expected nil result for volumeID %d, but got a result\n", tc.volumeID)
			} else {
				fmt.Printf("Detach successful for volumeID %d: %+v\n", tc.volumeID, result)
			}
		})
	}
}

func TestGetClusterId(t *testing.T) {
	apiKey := "ca172084-30c8-419d-9148-b88822a992d6"
	apiServer := "https://staging-infrahub-api.internal.ngbackend.cloud/v1"

	client := NewHyperstackClient(apiKey, apiServer, "test")
	hs := &Hyperstack{
		Client: client,
	}

	ctx := context.Background()
	testCases := []struct {
		name      string
		clusterID int
		expectNil bool
	}{
		{"valid_cluster_id", 1168, false},
		// {"valid_cluster_id", 2, true},
		// {"non_existent_id", 3, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Printf("Testing GetClusterId with clusterID: %d\n", tc.clusterID)
			result, err := hs.GetClusterDetail(ctx, tc.clusterID)
			if err != nil {
				t.Errorf("Error getting cluster ID: %v\n", err)
				return
			}
			fmt.Printf("Cluster ID: %+v\n", *result.Id)
			fmt.Printf("Cluster Name: %+v\n", *result.Name)
			fmt.Printf("Cluster Environment: %+v\n", *result.EnvironmentName)
		})
	}

}
