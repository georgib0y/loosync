package main

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"
)

type MockFile struct {
	name     string
	modified time.Time
	isDir    bool
	children map[string]*MockFile
}

func NewMockFile(name string) *MockFile {
	return &MockFile{name, time.Now(), false, map[string]*MockFile{}}
}

func NewMockDir(name string, children ...*MockFile) *MockFile {
	c := map[string]*MockFile{}

	for _, file := range children {
		c[file.Name()] = file
	}

	return &MockFile{name, time.Now(), true, c}
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
	if f.IsDir() {
		return fs.ModeDir
	}

	// regular file
	return fs.FileMode(0)
}

func (f MockFile) Type() fs.FileMode {
	return f.Mode().Type()
}

func (f MockFile) ModTime() time.Time {
	return f.modified
}

func (f MockFile) IsDir() bool {
	return f.isDir
}

func (f MockFile) Sys() any {
	return nil
}

func (f MockFile) AddChild(file *MockFile) error {
	if !f.IsDir() {
		return errors.New("Could not add child, file is not directory")
	}

	f.children[file.Name()] = file

	return nil
}

type MockFS struct {
	fsRoot *MockFile
}

func NewFilledMockFS() MockFS {
	return MockFS{
		NewMockDir(
			"/",
			NewMockFile("file1"),
			NewMockFile("file2"),
			NewMockDir(
				"subfolder",
				NewMockFile("file3"),
			),
		),
	}
}

func NewEmptyMockFS() MockFS {
	return MockFS{
		NewMockDir("/"),
	}
}

func NewMockFSWithFiles(children ...*MockFile) MockFS {
	return MockFS{NewMockDir("/", children...)}
}

func (f MockFS) FindFile(name string) (*MockFile, error) {
	return f.findFile(name, "/", f.fsRoot)
}

func (f MockFS) findFile(name string, p string, file *MockFile) (*MockFile, error) {
	if path.Join(p, file.Name()) == name {
		return file, nil
	}

	if file.IsDir() {
		for i := range file.children {
			if f1, err := f.findFile(name, path.Join(p, file.Name()), file.children[i]); err == nil {
				return f1, nil
			}
		}
	}

	return nil, fs.ErrNotExist
}

func (f MockFS) Open(name string) (fs.File, error) {
	file, err := f.FindFile(name)

	if err != nil {
		return nil, err
	}

	return file, nil
}

func (f MockFS) ReadDir(name string) ([]fs.DirEntry, error) {
	file, err := f.FindFile(name)

	if err != nil {
		return nil, err
	}

	if !file.IsDir() {
		return nil, fs.ErrNotExist
	}

	d := []fs.DirEntry{}

	for _, c := range file.children {
		d = append(d, c)
	}

	return d, nil
}

func (f MockFS) AddFile(p string, file *MockFile) error {
	dir, err := f.FindFile(p)

	if err != nil {
		return err
	}

	if !dir.IsDir() {
		return fmt.Errorf("Dir %s is not a dir", p)
	}

	dir.AddChild(file)

	return nil
}

func (f MockFS) Remove(p string) error {
	if p == f.fsRoot.Name() {
		return errors.New("Cannot remove root file")
	}

	dir, name := path.Split(p)
	file, err := f.FindFile(dir)
	if err != nil {
		return err
	}

	delete(file.children, name)
	return nil
}

func (f MockFS) Modify(p string) error {
	file, err := f.FindFile(p)

	if err != nil {
		return err
	}

	file.modified = time.Now()

	return nil
}

func TestMockFSFindsFile(t *testing.T) {
	fsys := NewFilledMockFS()

	files := map[string]*MockFile{
		"/":                fsys.fsRoot,
		"/file1":           fsys.fsRoot.children["file1"],
		"/subfolder":       fsys.fsRoot.children["subfolder"],
		"/subfolder/file3": fsys.fsRoot.children["subfolder"].children["file3"],
	}

	for name, file := range files {
		f, err := fsys.FindFile(name)

		if err != nil {
			t.Fatalf("Error finding file %s: %s", name, err)
		}

		if f != file {
			t.Fatalf("Wrong file found (expected %s, but got %s)", file.Name(), f.Name())
		}
	}
}

func TestMockFSWalksTree(t *testing.T) {
	fsys := NewFilledMockFS()
	visited := map[string]bool{}
	expected := map[string]bool{
		"/":                true,
		"/file1":           true,
		"/file2":           true,
		"/subfolder":       true,
		"/subfolder/file3": true,
	}

	fs.WalkDir(fsys, "/", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		t.Logf("visited path %s", path)
		visited[path] = true
		return nil
	})

	if !reflect.DeepEqual(visited, expected) {
		t.Fatalf("Visited paths not equal to expected paths")
	}
}

func TestMockFSAddFile(t *testing.T) {
	newFiles := []struct {
		insertDir string
		file      *MockFile
	}{
		{
			insertDir: "/",
			file:      NewMockFile("new1"),
		},
		{
			insertDir: "/",
			file:      NewMockFile("new2"),
		},
		{
			insertDir: "/",
			file:      NewMockDir("new_dir"),
		},
		{
			insertDir: "/new_dir",
			file:      NewMockFile("new3"),
		},
	}

	fs := NewEmptyMockFS()

	// add all new files
	for _, nF := range newFiles {
		err := fs.AddFile(nF.insertDir, nF.file)

		if err != nil {
			t.Fatalf("Could not add file: %s in dir %s", nF.file.Name(), nF.insertDir)
		}
	}

	// check all files have been added
	for _, nF := range newFiles {
		path := path.Join(nF.insertDir, nF.file.Name())
		file, err := fs.Open(path)

		if err != nil {
			t.Fatalf("Could not open file at %s, Err: %s", path, err)
		}

		if file != nF.file {
			t.Fatalf("File at path %s not equal to added file %s", path, nF.file.Name())
		}
	}
}

func TestMockFSRemovesFile(t *testing.T) {
	fsys := NewFilledMockFS()

	// delete file
	_, err := fsys.FindFile("/file1")
	if err != nil {
		t.Fatal("Could not find /file1")
	}

	if err := fsys.Remove("/file1"); err != nil {
		t.Fatalf("Error removing file: %s", err)
	}

	_, err = fsys.FindFile("/file1")
	if err != fs.ErrNotExist {
		t.Fatal("Did not get ErrNotExist error when finding removed file, got: ", err)
	}
}

func TestMockFSRemovesDir(t *testing.T) {
	fsys := NewFilledMockFS()

	// delete file
	_, err := fsys.FindFile("/subfolder")
	if err != nil {
		t.Fatal("Could not find /subfolder")
	}

	if err := fsys.Remove("/subfolder"); err != nil {
		t.Fatalf("Error removing file: %s", err)
	}

	_, err = fsys.FindFile("/subfolder")
	if err != fs.ErrNotExist {
		t.Error("Did not get ErrNotExist error when finding removed dir, got: ", err)
	}

	_, err = fsys.FindFile("/subfolder/file3")
	if err != fs.ErrNotExist {
		t.Error("Did not get ErrNotExist error when finding child file, got: ", err)
	}
}

// assumes that name is the full path to the entry in the fs
func findChild(root *node, name string) (*node, bool) {
	if name == root.name {
		return root, true
	}

	segments := strings.Split(name, "/")

	child := root
	for _, seg := range segments {
		if seg == root.name {
			continue
		}

		c, ok := child.children[seg]
		if !ok {
			return nil, false
		}

		child = c
	}

	return child, true
}

func TestNextSnapCreatesSnapshot(t *testing.T) {
	fsys := NewFilledMockFS()

	nodeEqFile := func(n *node, name string) bool {
		file, err := fsys.Open(name)
		if err != nil {
			t.Fatal("Could not open file: ", name)
		}
		stat, err := file.Stat()
		if err != nil {
			t.Fatal("Could not get stat for file: ", name)
		}
		return n.name == stat.Name() && n.isDir == stat.IsDir() && n.modTime == stat.ModTime()
	}

	p, err := NewPoller(fsys, "/")
	if err != nil {
		t.Fatal("Failed to create Poller", err)
	}

	root := p.nextSnap()

	if !nodeEqFile(root.children["file1"], "/file1") {
		t.Error("file1 does not exist or does not eq fs file")
	}
	if !nodeEqFile(root.children["file2"], "/file2") {
		t.Error("file2 does not exist or does not eq fs file")
	}
	if !nodeEqFile(root.children["subfolder"], "/subfolder") {
		t.Error("subfolder does not exist or does not eq fs file")
	}
	if !nodeEqFile(root.children["subfolder"].children["file3"], "/subfolder/file3") {
		t.Error("file3 does not exist or does not eq fs file")
	}
}

func TestPollerEmitsFSEvents(t *testing.T) {
	testCases := []struct {
		desc     string
		modFunc  func(f *MockFS) error
		expected map[string]Event
	}{
		{
			desc: "Create files",
			modFunc: func(f *MockFS) error {
				if err := f.AddFile("/", NewMockFile("newFile1")); err != nil {
					return err
				}
				if err := f.AddFile("/", NewMockFile("newFile2")); err != nil {
					return err
				}
				if err := f.AddFile("/subfolder", NewMockFile("newFile3")); err != nil {
					return err
				}
				return nil
			},
			expected: map[string]Event{
				"/newFile1":           {"/newFile1", CREATED},
				"/newFile2":           {"/newFile2", CREATED},
				"/subfolder/newFile3": {"/subfolder/newFile3", CREATED},
			},
		},
		{
			desc: "Modify files",
			modFunc: func(f *MockFS) error {
				if err := f.Modify("/file1"); err != nil {
					return err
				}

				if err := f.Modify("/subfolder/file3"); err != nil {
					return err
				}
				return nil
			},
			expected: map[string]Event{
				"/file1":           {"/file1", MODIFIED},
				"/subfolder/file3": {"/subfolder/file3", MODIFIED},
			},
		},
		{
			desc: "Remove files",
			modFunc: func(f *MockFS) error {
				if err := f.Remove("/file1"); err != nil {
					return err
				}
				if err := f.Remove("/subfolder"); err != nil {
					return err
				}
				return nil
			},
			expected: map[string]Event{
				"/file1":     {"/file1", DELETED},
				"/subfolder": {"/subfolder", DELETED},
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			fsys := NewFilledMockFS()
			p, err := NewPoller(fsys, "/")

			if err != nil {
				t.Error("Failed creating poller: ", err)
				return
			}

			// call diff once to populate the base snap
			p.diff()

			if err = tC.modFunc(&fsys); err != nil {
				t.Error("Failed modifying fsys: ", err)
				return
			}

			events := map[string]Event{}
			go func() {
				for p.events != nil && p.errors != nil {
					select {
					case event, ok := <-p.events:
						if !ok {
							p.events = nil
							continue
						}
						events[event.name] = event
					case err, ok := <-p.errors:
						if !ok {
							p.errors = nil
							continue
						}
						t.Error("Error while reading diffs: ", err)
					}
				}
			}()

			p.diff()
			time.Sleep(100 * time.Millisecond)
			p.Close()

			if !reflect.DeepEqual(tC.expected, events) {
				t.Errorf("Result:\n%s\n did not eq expected:\n%s", events, tC.expected)
			}
		})
	}
}
