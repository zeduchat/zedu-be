package minio

import (
	"fmt"
	"mime/multipart"

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

// Scan file with ClamAV before uploading
func ScanFileWithClamAV(file multipart.File) error {
	if ClamAV == nil {
		return fmt.Errorf("ClamAV is not initialized. Line 36")
	}

	response, err := ClamAV.ScanStream(file, make(chan bool)) // Use global clamAV instance
	if err != nil {
		return fmt.Errorf("ClamAV scan failed: %v", err)
	}

	for result := range response {
		if result.Status == clamd.RES_FOUND || result.Status == "FOUND" {
			return fmt.Errorf("malware detected: %s", result.Description)
		}
	}

	return nil
}