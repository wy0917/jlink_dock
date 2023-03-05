package model

import (
	"mime/multipart"
	"time"
)

type UploadedFiles struct {
	elf    *multipart.FileHeader `form:"elf"`    // ELF file to flash onto the board
	script *multipart.FileHeader `form:"script"` // Script.zip, which will be run
}

type APIError struct {
	ErrorCode    int
	ErrorMessage string
	CreatedAt    time.Time
}
