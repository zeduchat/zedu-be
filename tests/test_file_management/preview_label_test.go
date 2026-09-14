package test_file_management

import (
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
)

func TestBuildPreviewMessagePDF(t *testing.T) {
	t.Run("PDF file attachment with zip MimeType returns PDF label", func(t *testing.T) {
		media := []models.File{
			{
				FileName: "document_sample.pdf",
				FileType: "pdf",
				MimeType: "application/zip", // Content sniffing misdetected zip stream
			},
		}

		preview := models.BuildPreviewMessage("", media)
		if preview != "📄 PDF" {
			t.Errorf("BuildPreviewMessage() = %q; want %q", preview, "📄 PDF")
		}
	})

	t.Run("Word document returns Document label", func(t *testing.T) {
		media := []models.File{
			{
				FileName: "contract.docx",
				FileType: "docx",
				MimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			},
		}

		preview := models.BuildPreviewMessage("", media)
		if preview != "📝 Document" {
			t.Errorf("BuildPreviewMessage() = %q; want %q", preview, "📝 Document")
		}
	})
}
