package mock

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"
	"time"
)

type MockFile struct {
	name     string
	content  []byte
	modified time.Time
	isDir    bool
	children map[string]*MockFile
}

func NewMockFile(name string) *MockFile {
	content := []byte(fmt.Sprintf("The Name of this file is %s", name))
	return &MockFile{name, content, time.Now(), false, map[string]*MockFile{}}
}

func NewMockDir(name string, children ...*MockFile) *MockFile {
	c := map[string]*MockFile{}

	for _, file := range children {
		c[file.Name()] = file
	}

	return &MockFile{name, []byte{}, time.Unix(0, 0), true, c}

}

func (f MockFile) Stat() (fs.FileInfo, error) {
	return f, nil
}

func (f MockFile) Info() (fs.FileInfo, error) {
	return f, nil
}

func (f *MockFile) Read(p []byte) (int, error) {
	n := copy(p, f.content)

	return n, io.EOF
}

func (f *MockFile) Write(p []byte) (int, error) {
	c := make([]byte, len(p))
	n := copy(c, p)
	f.content = c

	return n, nil
}

func (f MockFile) Close() error {
	return nil
}

func (f MockFile) Name() string {
	return f.name
}

func (f MockFile) Size() int64 {
	return int64(len(f.content))
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
			".",
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
		NewMockDir("."),
	}
}

func NewMockFSWithFiles(children ...*MockFile) MockFS {
	return MockFS{NewMockDir(".", children...)}
}

func (f MockFS) FindFile(p string) (*MockFile, error) {
	if p == "." || p == "./" {
		return f.fsRoot, nil
	}

	// remove ./ from any paths
	if strings.HasPrefix(p, "./") {
		p = p[2:]
	}

	return f.findFile(p, "", f.fsRoot)
}

func (f MockFS) findFile(p string, currP string, file *MockFile) (*MockFile, error) {
	if path.Join(currP, file.Name()) == p {
		return file, nil
	}

	if file.IsDir() {
		for i := range file.children {
			if f1, err := f.findFile(p, path.Join(currP, file.Name()), file.children[i]); err == nil {
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
	if p == "." || p == "./" {
		return errors.New("Cannot remove root file")
	}

	if !strings.HasPrefix(p, "./") {
		p = fmt.Sprintf("./%s", p)
	}

	dir, name := path.Split(p)
	parent, err := f.FindFile(dir)
	if err != nil {
		return err
	}

	delete(parent.children, name)
	return nil
}
