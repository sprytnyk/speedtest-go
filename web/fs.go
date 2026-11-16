package web

import (
	"io"
	"net/http"
	"os"
)

// Credit: https://stackoverflow.com/questions/49589685/good-way-to-disable-directory-listing-with-http-fileserver-in-go
type justFilesFilesystem struct {
	fs               http.FileSystem
	readDirBatchSize int
}

func (fs justFilesFilesystem) Open(name string) (http.File, error) {
	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}
	return neuteredStatFile{File: f, readDirBatchSize: fs.readDirBatchSize}, nil
}

type neuteredStatFile struct {
	http.File
	readDirBatchSize int
}

func (e neuteredStatFile) Stat() (os.FileInfo, error) {
	s, err := e.File.Stat()
	if err != nil {
		return nil, err
	}
	if !s.IsDir() {
		return s, nil
	}
	for {
		files, err := e.File.Readdir(e.readDirBatchSize)
		switch err {
		case io.EOF:
			return nil, os.ErrNotExist
		case nil:
			for _, f := range files {
				if f.Name() == "index.html" {
					return s, nil
				}
			}
		default:
			return nil, err
		}
	}
}
