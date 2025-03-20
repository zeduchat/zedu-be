package models

import "mime/multipart"

type UploadRequest struct {
	Files []*multipart.FileHeader `form:"files" binding:"required"` // Multiple file headers
}
