package main

import (
	"fmt"

	"github.com/georib0y/loosync/mock"

	"testing"
)

// modFSFunc are inverse of what is in the description
// Deleted Files add files to the fsys in mod func so that the diff can pick up
// on what files need to be deleted
// Likewise to the Created Files, which will remove files to see which files
// need to be created
func TestDiffing(t *testing.T) {
	testCases := []struct {
		desc      string
		modFsFunc func(*mock.MockFS)
		expected  map[string]Diff
	}{
		{
			desc: "Deleted Files",
			modFsFunc: func(fsys *mock.MockFS) {
				if err := fsys.AddFile(".", mock.NewMockFile("newFile1")); err != nil {
					t.Fatalf("Could not add file %s, %s", "newFile1", err)
				}
				if err := fsys.AddFile(".", mock.NewMockFile("newFile2")); err != nil {
					t.Fatalf("Could not add file %s, %s", "newFile2", err)
				}
				if err := fsys.AddFile("subfolder", mock.NewMockFile("newFile3")); err != nil {
					t.Fatalf("Could not add file %s, %s", "newFile3", err)
				}
			},
			expected: map[string]Diff{
				"newFile1":           {"newFile1", DELETED},
				"newFile2":           {"newFile2", DELETED},
				"subfolder/newFile3": {"subfolder/newFile3", DELETED},
			},
		},
		{
			desc: "Modified Files",
			modFsFunc: func(fsys *mock.MockFS) {
				for _, name := range []string{"file1", "subfolder/file3"} {
					f, err := fsys.Open(name)
					if err != nil {
						t.Fatalf("Could not open file %s to modify, %s", name, err)
					}
					fmt.Fprintf(f.(*mock.MockFile), "This is a modified %s", name)
				}
			},
			expected: map[string]Diff{
				"file1":           {"file1", MODIFIED},
				"subfolder/file3": {"subfolder/file3", MODIFIED},
			},
		},
		{
			desc: "Created Files",
			modFsFunc: func(fsys *mock.MockFS) {
				if err := fsys.Remove("file1"); err != nil {
					t.Fatalf("Could not remove file %s, %s", "file1", err)
				}
				if err := fsys.Remove("subfolder"); err != nil {
					t.Fatalf("Could not remove folder %s, %s", "subfolder", err)
				}
			},
			expected: map[string]Diff{
				"file1":           {"file1", CREATED},
				"subfolder":       {"subfolder", CREATED},
				"subfolder/file3": {"subfolder/file3", CREATED},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			fsys := mock.NewFilledMockFS()
			s1, err := NewFSSnap(fsys)
			if err != nil {
				t.Fatal(err)
			}

			tc.modFsFunc(&fsys)
			s2, err := NewFSSnap(fsys)
			if err != nil {
				t.Fatal(err)
			}

			diffs := DiffSnaps(s1, s2)

			if len(diffs) != len(tc.expected) {
				t.Fatalf("Did not get expected number of diffs, got: %d, expected: %d", len(diffs), len(tc.expected))
			}

			for _, diff := range diffs {
				expDiff, ok := tc.expected[diff.name]
				if !ok {
					t.Errorf("Could not find expected diff with name %s", diff.name)
					continue
				}
				if expDiff != diff {
					t.Errorf("Wrong diff. Got %s, expected: %s", diff, expDiff)
				}
			}
		})
	}

}
