package models

import "mime/multipart"

type UploadRequest struct {
	Files []*multipart.FileHeader `form:"files" binding:"required"` // Multiple file headers
}

type FileType struct {
	MimeType string `json:"mime_type"`
	Category string `json:"category"`
}

// AllowedMimeTypes defines commonly used MIME types grouped by their category.
var AllowedFileTypes = []FileType{
	// Images
	{"image/png", "images/"},
	{"image/jpeg", "images/"},
	{"image/jpg", "images/"},
	{"image/gif", "images/"},
	{"image/webp", "images/"},

	// Documents
	{"text/csv", "documents/"},           // .csv
	{"application/pdf", "documents/"},    // .pdf
	{"application/msword", "documents/"}, // .doc (legacy Word)
	{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", "documents/"},   // .docx
	{"application/vnd.ms-word.document.macroEnabled.12", "documents/"},                          // .docm (macro-enabled Word)
	{"application/x-msword", "documents/"},                                                      // Alternative .doc MIME
	{"application/zip", "documents/"},                                                           // Some .docx files are detected as ZIP
	{"application/vnd.ms-excel", "documents/"},                                                  // .xls (legacy Excel)
	{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "documents/"},         // .xlsx
	{"application/vnd.ms-powerpoint", "documents/"},                                             // .ppt (legacy PowerPoint)
	{"application/vnd.openxmlformats-officedocument.presentationml.presentation", "documents/"}, // .pptx
	{"text/plain", "documents/"},                                                                // .txt (plain text)

	// Audio
	{"audio/mpeg", "audio/"}, // .mp3
	{"audio/wav", "audio/"},  // .wav

	// Video
	{"video/mp4", "videos/"},  // .mp4
	{"video/webm", "videos/"}, // .webm
}
