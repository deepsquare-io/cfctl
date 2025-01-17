package analytics

import (
	"crypto/md5"
	"encoding/hex"
	"os"

	"github.com/denisbrodbeck/machineid"
)

// MachineID returns protected id for the current machine
func MachineID() (string, error) {
	id, err := machineid.ProtectedID("clusterfactory-cfctl")
	if err != nil {
		return MachineIDFromHostname()
	}
	return id, err
}

// MachineIDFromHostname generates a machine id hash from hostname
func MachineIDFromHostname() (string, error) {
	name, err := os.Hostname()
	if err != nil {
		return "", err
	}
	sum := md5.Sum([]byte(name))
	return hex.EncodeToString(sum[:]), nil
}
