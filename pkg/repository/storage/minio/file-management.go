package minio

import (
	"fmt"

	"github.com/dutchcoders/go-clamd"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)

var ClamAV *clamd.Clamd // Global ClamAV instance

// Initialize ClamAv
func ConnectToClamAV(logger *utility.Logger, clamav config.Clamav) *clamd.Clamd {
	clamavHost := clamav.ClamavHost
	clamavPort := clamav.ClamavPort

	if clamavHost == "" || clamavPort == "" {
		utility.LogAndPrint(logger, fmt.Sprintf("ClamAV environment variables not set correctly"))
		return nil
	}

	clamavAddress := fmt.Sprintf("tcp://%s:%s", clamavHost, clamavPort)
	utility.LogAndPrint(logger, fmt.Sprintf("ClamAV initialized on: %v", clamavAddress))

	ClamAV := clamd.NewClamd(clamavAddress)
	utility.LogAndPrint(logger, fmt.Sprintf("ClamAV as seen: %v", ClamAV))

	return ClamAV
}
