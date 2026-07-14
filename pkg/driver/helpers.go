package driver

import "github.com/NexGenCloud/hyperstack-sdk-go/lib/volume"

// findVolumeAttachment returns the first attachment record whose InstanceId
// matches instanceID, or nil if no match is found.
func findVolumeAttachment(attachments *[]volume.AttachmentsFieldsForVolume, instanceID int) *volume.AttachmentsFieldsForVolume {
	if attachments == nil {
		return nil
	}
	for i := range *attachments {
		a := &(*attachments)[i]
		if a.InstanceId != nil && *a.InstanceId == instanceID {
			return a
		}
	}
	return nil
}
