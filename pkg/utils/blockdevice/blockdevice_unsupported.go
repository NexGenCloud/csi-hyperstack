//go:build !linux
// +build !linux

package blockdevice

import (
	"errors"
)

func IsBlockDevice(path string) (bool, error) {
	return false, errors.New("IsBlockDevice is not implemented for this OS")
}

func GetBlockDeviceSize(path string) (int64, error) {
	return -1, errors.New("GetBlockDeviceSize is not implemented for this OS")
}

func RescanBlockDeviceGeometry(devicePath string, deviceMountPath string, newSize int64) error {
	return errors.New("RescanBlockDeviceGeometry is not implemented for this OS")
}
