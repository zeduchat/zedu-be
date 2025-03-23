package models

import "mime/multipart"

type UploadRequest struct {
	Files []*multipart.FileHeader `form:"files" binding:"required"` // Multiple file headers
}

type UploadedFileResponse struct {
	FileName string `json:"file_name"`
	FileType string `json:"file_type"`
	MimeType string `json:"mime_type"`
	FileLink string `json:"file_link"`
}

type FileType struct {
	MimeType string `json:"mime_type"`
	Category string `json:"category"`
}

// AllowedMimeTypes defines commonly used MIME types grouped by their category.
var AllowedFileTypes = []FileType{
	// Images
	{"image/png", "file-uploads/images/"},
	{"image/jpeg", "file-uploads/images/"},
	{"image/jpg", "file-uploads/images/"},
	{"image/gif", "file-uploads/images/"},
	{"image/webp", "file-uploads/images/"},

	// Documents
	{"text/plain", "file-uploads/documents/"},                                                                // .csv
	{"text/csv", "file-uploads/documents/"},                                                                  // .csv
	{"application/pdf", "file-uploads/documents/"},                                                           // .pdf
	{"application/msword", "file-uploads/documents/"},                                                        // .doc (legacy Word)
	{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", "file-uploads/documents/"},   // .docx
	{"application/vnd.ms-word.document.macroEnabled.12", "file-uploads/documents/"},                          // .docm (macro-enabled Word)
	{"application/x-msword", "file-uploads/documents/"},                                                      // Alternative .doc MIME
	{"application/zip", "file-uploads/documents/"},                                                           // Some .docx files are detected as ZIP
	{"application/vnd.ms-excel", "file-uploads/documents/"},                                                  // .xls (legacy Excel)
	{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "file-uploads/documents/"},         // .xlsx
	{"application/vnd.ms-powerpoint", "file-uploads/documents/"},                                             // .ppt (legacy PowerPoint)
	{"application/vnd.openxmlformats-officedocument.presentationml.presentation", "file-uploads/documents/"}, // .pptx

	// Audio
	{"audio/mpeg", "file-uploads/audio/"}, // .mp3
	{"audio/wav", "file-uploads/audio/"},  // .wav

	// Video
	{"video/mp4", "file-uploads/videos/"},  // .mp4
	{"video/webm", "file-uploads/videos/"}, // .webm
}
