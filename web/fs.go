package web

import (
	"io"
	"net/http"
	"os"
)

// noListingFS lets the file server serve directory indexes without listing directories.
// Credit: https://stackoverflow.com/questions/49589685/good-way-to-disable-directory-listing-with-http-fileserver-in-go
type noListingFS struct{ http.FileSystem }

func (fs noListingFS) Open(name string) (http.File, error) {
	file, err := fs.FileSystem.Open(name)
	if err != nil {
		return nil, err
	}
	return noListingFile{file}, nil
}

type noListingFile struct{ http.File }

func (f noListingFile) Stat() (os.FileInfo, error) {
	info, err := f.File.Stat()
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return info, nil
	}

	for {
		entries, err := f.Readdir(2)
		if err == io.EOF {
			return nil, os.ErrNotExist
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.Name() == "index.html" {
				return info, nil
			}
		}
	}
}
