package mock

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"reflect"
	"testing"
)

func TestMockFileRead(t *testing.T) {
	filename := "name"

	f := NewMockFile(filename)
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	s := string(b)
	t.Logf("Read: %s", s)

	exp := fmt.Sprintf("The Name of this file is %s", filename)

	if s != exp {
		t.Fatalf("Did not get expected output from read, got: \"%s\" expected: \"%s\"", s, exp)
	}
}

func TestMockFSFindsFile(t *testing.T) {
	fsys := NewFilledMockFS()

	files := map[string]*MockFile{
		".":               fsys.fsRoot,
		"file1":           fsys.fsRoot.children["file1"],
		"subfolder":       fsys.fsRoot.children["subfolder"],
		"subfolder/file3": fsys.fsRoot.children["subfolder"].children["file3"],
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
		".":               true,
		"file1":           true,
		"file2":           true,
		"subfolder":       true,
		"subfolder/file3": true,
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
			insertDir: ".",
			file:      NewMockFile("new1"),
		},
		{
			insertDir: ".",
			file:      NewMockFile("new2"),
		},
		{
			insertDir: ".",
			file:      NewMockDir("new_dir"),
		},
		{
			insertDir: "new_dir",
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
	_, err := fsys.FindFile("file1")
	if err != nil {
		t.Fatal("Could not find file1")
	}

	if err := fsys.Remove("file1"); err != nil {
		t.Fatalf("Error removing file: %s", err)
	}

	_, err = fsys.FindFile("file1")
	if err != fs.ErrNotExist {
		t.Fatal("Did not get ErrNotExist error when finding removed file, got: ", err)
	}
}

func TestMockFSRemovesDir(t *testing.T) {
	fsys := NewFilledMockFS()

	// delete file
	_, err := fsys.FindFile("subfolder")
	if err != nil {
		t.Fatal("Could not find subfolder")
	}

	if err := fsys.Remove("subfolder"); err != nil {
		t.Fatalf("Error removing file: %s", err)
	}

	_, err = fsys.FindFile("subfolder")
	if err != fs.ErrNotExist {
		t.Error("Did not get ErrNotExist error when finding removed dir, got: ", err)
	}

	_, err = fsys.FindFile("subfolder/file3")
	if err != fs.ErrNotExist {
		t.Error("Did not get ErrNotExist error when finding child file, got: ", err)
	}
}

func TestNormalFSDotReturnsCurrentFile(t *testing.T) {
	dir := "/tmp"
	fsys := os.DirFS(dir).(fs.ReadDirFS)

	dirEnts, err := fsys.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	f, err := fsys.Open(".")
	if err != nil {
		t.Fatal(err)
	}

	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Root dir name is %s", info.Name())

	read, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Root dir read() is: %s", read)

	osDirEnts, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(osDirEnts) == 0 {
		t.Skipf("Did not find any dir entries for %s", dir)
	}

	if len(dirEnts) != len(osDirEnts) {
		t.Fatalf("Len of dir ents (%d) not equal to os dir ents (%d)", len(dirEnts), len(osDirEnts))
	}

	t.Logf("There are %d entries in %s", len(dirEnts), dir)
}
