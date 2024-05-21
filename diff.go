package main

import (
	"fmt"
	"io/fs"
	"path"
	"time"
)

type Snap struct {
	name     string
	size     int64
	isDir    bool
	modTime  time.Time
	children map[string]*Snap
}

func newFromFileInfo(info fs.FileInfo) *Snap {
	return &Snap{
		name:     info.Name(),
		size:     info.Size(),
		isDir:    info.IsDir(),
		modTime:  info.ModTime(),
		children: map[string]*Snap{},
	}
}

func NewFSSnap(fsys fs.ReadDirFS) (*Snap, error) {
	rootInfo, err := fs.Stat(fsys, ".")
	if err != nil {
		return nil, err
	}

	snap := newFromFileInfo(rootInfo)
	err = addSnapChildren(fsys, ".", snap)
	return snap, err
}

func addSnapChildren(fsys fs.ReadDirFS, p string, s *Snap) error {
	ents, err := fsys.ReadDir(p)
	if err != nil {
		return err
	}

	for _, e := range ents {
		info, err := e.Info()
		if err != nil {
			return err
		}

		child := newFromFileInfo(info)

		if child.isDir {
			addSnapChildren(fsys, path.Join(p, child.name), child)
		}

		s.children[child.name] = child
	}

	return nil
}

// gets the names of the chidren from s that aren't in o
func (s *Snap) chidrenAbsent(o *Snap) []*Snap {
	absent := []*Snap{}

	for name, child := range s.children {
		if _, ok := o.children[name]; !ok {
			absent = append(absent, child)
		}
	}

	return absent
}

// gets the names of the chidren from s that are common with o
func (s *Snap) namesInCommon(o *Snap) []string {
	common := []string{}

	for name := range s.children {
		if _, ok := o.children[name]; ok {
			common = append(common, name)
		}
	}

	return common
}

func (s *Snap) isModified(o *Snap) bool {
	return s.size != o.size
}

type DiffType int

const (
	CREATED DiffType = iota
	MODIFIED
	DELETED
)

func (d DiffType) String() string {
	switch d {
	case CREATED:
		return "CREATED"
	case MODIFIED:
		return "MODIFIED"
	case DELETED:
		return "DELETED"
	default:
		panic(fmt.Sprintf("Unknown diff type %d", d))
	}
}

type Diff struct {
	name string
	t    DiffType
}

func (d Diff) String() string {
	return fmt.Sprintf("Name: %s, Type: %s", d.name, d.t)
}

// Compares two snaps, outputs a slice of diffs from the perspective of base
// eg if a file exists in base but does not exist in comp, then a CREATED diff
// will be made, and if a file exists in comp but on base then the a DELETED
// diff will be made
//
// The function aims to output diffs in an order that is convinient to process
// sequentially (CREATED diffs occur as parent directories and then leaf nodes,
// and DELETED events occur as leaf nodes then their directories)
func DiffSnaps(base, comp *Snap) []Diff {
	dChan := make(chan Diff)

	go func() {
		diffSnaps(base, comp, ".", dChan)
		close(dChan)
	}()

	diffs := []Diff{}
	for d := range dChan {
		diffs = append(diffs, d)
	}

	return diffs
}

func diffSnaps(base, comp *Snap, p string, d chan Diff) {
	for _, created := range base.chidrenAbsent(comp) {
		markAsCreated(created, p, d)
	}

	for _, removed := range comp.chidrenAbsent(base) {
		markAsDeleted(removed, p, d)
	}

	for _, common := range base.namesInCommon(comp) {
		c1, c2 := base.children[common], comp.children[common]

		if c1.isDir && c2.isDir {
			diffSnaps(c1, c2, path.Join(p, common), d)
			return
		}

		if c1.isModified(c2) {
			d <- Diff{path.Join(p, common), MODIFIED}
		}
	}
}

func markAsCreated(s *Snap, p string, d chan Diff) {
	p = path.Join(p, s.name)
	d <- Diff{p, CREATED}
	for _, c := range s.children {
		markAsCreated(c, p, d)
	}
}

func markAsDeleted(s *Snap, p string, d chan Diff) {
	p = path.Join(p, s.name)
	for _, c := range s.children {
		markAsDeleted(c, p, d)
	}
	d <- Diff{p, DELETED}
}
