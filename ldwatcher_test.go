package main

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"path"
	"reflect"
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
			t.Errorf("Error finding file %s: %s", name, err)
			return
		}

		if f != file {
			t.Errorf("Wrong file found (expected %s, but got %s)", file.Name(), f.Name())
			return
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
		t.Errorf("Visited paths not equal to expected paths")
		return
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
			t.Errorf("Could not add file: %s in dir %s", nF.file.Name(), nF.insertDir)
			return
		}
	}

	// check all files have been added
	for _, nF := range newFiles {
		path := path.Join(nF.insertDir, nF.file.Name())
		file, err := fs.Open(path)

		if err != nil {
			t.Errorf("Could not open file at %s, Err: %s", path, err)
			return
		}

		if file != nF.file {
			t.Errorf("File at path %s not equal to added file %s", path, nF.file.Name())
			return
		}
	}
}

func TestMockFSRemovesFile(t *testing.T) {
	fsys := NewFilledMockFS()

	// delete file
	_, err := fsys.FindFile("/file1")
	if err != nil {
		t.Error("Could not find /file1")
	}

	if err := fsys.Remove("/file1"); err != nil {
		t.Errorf("Error removing file: %s", err)
		return
	}

	_, err = fsys.FindFile("/file1")
	if err != fs.ErrNotExist {
		t.Error("Did not get ErrNotExist error when finding removed file, got: ", err)
	}
}

func TestMockFSRemovesDir(t *testing.T) {
	fsys := NewFilledMockFS()

	// delete file
	_, err := fsys.FindFile("/subfolder")
	if err != nil {
		t.Error("Could not find /subfolder")
	}

	if err := fsys.Remove("/subfolder"); err != nil {
		t.Errorf("Error removing file: %s", err)
		return
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

func TestLDSnapFromFS(t *testing.T) {
	fsys := NewFilledMockFS()

	s, err := NewLDSnap(fsys, "/")

	if err != nil {
		t.Error("Failed to create LDSnap: ", err)
		return
	}

	fs.WalkDir(fsys, "/", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Error("Walk dir failed with err: ", err)
			return err
		}

		n, err := s.FindNode(path)

		if err != nil {
			return err
		}

		if n.Name() != d.Name() {
			err := fmt.Errorf("Node name %s and DirEntry name %s are not eq", n.Name(), d.Name())
			t.Error(err)
			return err
		}

		if n.IsDir() != d.IsDir() {
			err := fmt.Errorf("Node.IsDir() == %t and DirEntry.IsDir == %t are not eq", n.IsDir(), d.IsDir())
			t.Error(err)
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		if info.ModTime() != n.ModTime() {
			err := fmt.Errorf("Node mod time %s and DirEntry mod time %s are not eq", n.ModTime(), info.ModTime())
			t.Error(err)
			return err
		}

		return nil
	})
}

func TestEqualLDSnapsHaveNoDiffs(t *testing.T) {
	fsys := NewFilledMockFS()

	// two equal snaps
	s1, err := NewLDSnap(fsys, "/")
	if err != nil {
		t.Error("Could not create snap: ", err)
		return
	}

	s2, _ := NewLDSnap(fsys, "/")

	diffs := make(chan LDDiff, MAX_DIFFS)

	DiffSnaps(s1, s2, diffs)

	if len(diffs) > 0 {
		t.Error("Differences found on two equal snaps")
	}
}

func TestDiffSnapsCreateLDDiffs(t *testing.T) {
	testCases := []struct {
		desc     string
		modFunc  func(f *MockFS) error
		expected map[string]LDDiffType
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
			expected: map[string]LDDiffType{
				"newFile1": CREATED,
				"newFile2": CREATED,
				"newFile3": CREATED,
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
			expected: map[string]LDDiffType{
				"file1": MODIFIED,
				"file3": MODIFIED,
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
			expected: map[string]LDDiffType{
				"file1":     DELETED,
				"subfolder": DELETED,
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			fsys := NewFilledMockFS()
			s1, err := NewLDSnap(fsys, "/")

			if err != nil {
				t.Error("Failed creating base snap: ", err)
				return
			}

			if err = tC.modFunc(&fsys); err != nil {
				t.Error("Failed modifying fsys: ", err)
				return
			}

			s2, err := NewLDSnap(fsys, "/")
			if err != nil {
				t.Error("Failed creating second snap: ", err)
				return
			}

			diffs := make(chan LDDiff, MAX_DIFFS)
			go DiffSnaps(s1, s2, diffs)

			res := map[string]LDDiffType{}
			for diff := range diffs {
				res[diff.node.name] = diff.dType
			}

			if !reflect.DeepEqual(tC.expected, res) {
				resStr := diffMapToString(res)
				expStr := diffMapToString(tC.expected)
				t.Errorf("Result:\n%s\n did not eq expected:\n%s", resStr, expStr)
			}
		})
	}
}

func diffMapToString(diffs map[string]LDDiffType) string {
	out := "{\n"
	for k, v := range diffs {
		dTypeStr := "UNKNOWN"
		switch v {
		case CREATED:
			dTypeStr = "CREATED"
		case MODIFIED:
			dTypeStr = "MODIFIED"
		case DELETED:
			dTypeStr = "DELETED"

		}

		out += fmt.Sprintf("\t{name: %s, dType: %s}\n", k, dTypeStr)
	}

	return out + "}"
}

// func TestCompareLDSnaps(t *testing.T) {
// 	baseSnap := NewLDSnap(NewFilledMockFS())

// }

// func TestNewLDWatcherCreatesLDSnap(t *testing.T) {
// 	fsys := NewFilledMockFS()

// 	ldw := NewLDWatcher(fsys)

// }
