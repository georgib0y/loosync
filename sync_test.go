package main

import (
	"io/fs"
	"log"
	"math/rand"
	"testing"
	"time"
)

var MockFSTree = map[string]MockFile{
	"/":                {"/", true},
	"/file1":           {"file1", false},
	"/file2":           {"file2", false},
	"/subfolder":       {"subfolder", true},
	"/subfolder/file3": {"file3", false},
}

type MockFile struct {
	name  string
	isDir bool
}

func (f MockFile) Stat() (fs.FileInfo, error) {
	return f, nil
}

func (f MockFile) Info() (fs.FileInfo, error) {
	return f, nil
}

func (f MockFile) Read(p []byte) (int, error) {
	return 0, nil
}

func (f MockFile) Close() error {
	return nil
}

func (f MockFile) Name() string {
	return f.name
}

func (f MockFile) Size() int64 {
	return int64(rand.Uint32())
}

func (f MockFile) Mode() fs.FileMode {
	if f.isDir {
		return fs.ModeDir
	}

	// regular file
	return fs.FileMode(0)
}

func (f MockFile) Type() fs.FileMode {
	return f.Mode().Type()
}

func (f MockFile) ModTime() time.Time {
	return time.Now()
}

func (f MockFile) IsDir() bool {
	return f.isDir
}

func (f MockFile) Sys() any {
	return nil
}

type MockFS struct {
	fstree map[string]MockFile
}

func NewMockFS() MockFS {
	return MockFS{
		map[string]MockFile{
			"/":                {"/", true},
			"/file1":           {"file1", false},
			"/file2":           {"file2", false},
			"/subfolder":       {"subfolder", true},
			"/subfolder/file3": {"file3", false},
		}}
}

func (f MockFS) Open(name string) (fs.File, error) {
	log.Printf("visited %s\n", name)
	file := MockFSTree[name]

	if (MockFile{}) == file {
		return nil, fs.ErrNotExist
	}

	return file, nil
}

func (f MockFS) ReadDir(name string) ([]fs.DirEntry, error) {
	switch name {
	case "/":
		return []fs.DirEntry{MockFSTree["/file1"], MockFSTree["/file2"], MockFSTree["/subfolder"]}, nil

	case "/subfolder":
		return []fs.DirEntry{MockFSTree["/subfolder/file3"]}, nil

	default:
		return []fs.DirEntry{}, nil
	}
}

func TestMockFSWalksTree(t *testing.T) {
	fsys := NewMockFS()
	visited := map[string]bool{}

	fs.WalkDir(fsys, "/", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		t.Logf("visited path %s", path)
		visited[path] = true
		return nil
	})

	for path := range fsys.fstree {
		if !visited[path] {
			t.Errorf("path %s not visited", path)
		}
	}
}
