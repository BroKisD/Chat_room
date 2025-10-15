package filetransfer

import (
	"io"
	"os"
	"path/filepath"
)

type FileTransfer struct {
	uploadDir string
}

func New(uploadDir string) *FileTransfer {
	return &FileTransfer{
		uploadDir: uploadDir,
	}
}

func (ft *FileTransfer) Upload(filename string, content io.Reader) error {
	path := filepath.Join(ft.uploadDir, filename)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, content)
	return err
}

func (ft *FileTransfer) Download(filename string) (io.ReadCloser, error) {
	path := filepath.Join(ft.uploadDir, filename)
	return os.Open(path)
}

func (ft *FileTransfer) UploadDir() string {
	return ft.uploadDir
}
