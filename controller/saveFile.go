package controller

import (
	"archive/zip"
	"bytes"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
)

const elfClassARM = 0x28

func isArmElf(b []byte) bool {
	if len(b) < 4 {
		return false
	}

	if !bytes.Equal(b[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		return false
	}

	class := b[18]
	return class == elfClassARM
}

func checkElf(file *multipart.FileHeader) error {
	elfFileHandle, err := file.Open()
	if err != nil {
		return err
	}
	defer elfFileHandle.Close()

	var b bytes.Buffer
	_, err = io.Copy(&b, elfFileHandle)
	if err != nil {
		return err
	}

	if !isArmElf(b.Bytes()) {
		return err
	}

	return nil
}

func ensureDirExist(dir string) error {
	// Check if the directory already exists
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		// If the directory does not exist, create it
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			log.Printf("Failed to create directory: %v", err)
			return err
		}
		log.Printf("Created directory: %v", dir)
	} else if err != nil {
		log.Printf("Error checking if directory exists: %v", err)
		return err
	}

	return nil
}

func extractZipFile(file *multipart.FileHeader, targetDir string) error {
	err := ensureDirExist(targetDir)
	if err != nil {
		return err
	}

	// Open the uploaded file
	uploadedFile, err := file.Open()
	if err != nil {
		return err
	}
	defer uploadedFile.Close()

	// Check if the uploaded file is a ZIP file
	zipReader, err := zip.NewReader(uploadedFile, file.Size)
	if err != nil {
		return err
	}

	// Extract the contents of the ZIP file to the target directory
	for _, zipFile := range zipReader.File {
		filename := filepath.Join(targetDir, zipFile.Name)

		// Create the corresponding directory in the target directory
		if zipFile.FileInfo().IsDir() {
			if err := os.MkdirAll(filename, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		// Create the corresponding file in the target directory
		destFile, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, zipFile.Mode())
		if err != nil {
			return err
		}

		// Extract the contents of the file to the corresponding file in the target directory
		srcFile, err := zipFile.Open()
		if err != nil {
			destFile.Close()
			return err
		}

		_, err = io.Copy(destFile, srcFile)
		destFile.Close()
		srcFile.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
