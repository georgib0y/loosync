package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"time"
)

type FileSyncState struct {
	name     string
	modified time.Time
	isDir    bool
}

func (s FileSyncState) String() string {
	return fmt.Sprintf("Name: %s, Modified: %s, isDir: %t", s.name, s.modified, s.isDir)
}

func fromDirEntry(d fs.DirEntry) (FileSyncState, error) {
	info, err := d.Info()

	if err != nil {
		return FileSyncState{}, err
	}

	return FileSyncState{
		name:     d.Name(),
		modified: info.ModTime(),
		isDir:    d.IsDir(),
	}, nil
}

func initSync(root string) {
	files := map[string]FileSyncState{}
	fsys := os.DirFS(root)

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		s, err := fromDirEntry(d)

		if err != nil {
			return err
		}

		files[path] = s

		return nil
	})

	if err != nil {
		panic(err)
	}

	for _, f := range files {
		log.Println(f)
	}
}
