package main

import (
	"fmt"
	"io/fs"
	"path"
	"time"
)

const MAX_DIFFS int = 1024

type LDSnapNode struct {
	name     string
	isDir    bool
	modTime  time.Time
	children map[string]*LDSnapNode
}

func (n LDSnapNode) Name() string {
	return n.name
}

func (n LDSnapNode) IsDir() bool {
	return n.isDir
}

func (n LDSnapNode) ModTime() time.Time {
	return n.modTime
}

type LDSnap struct {
	root  *LDSnapNode
	rPath string
}

func NewLDSnap(fsys fs.ReadDirFS, path string) (LDSnap, error) {
	rNode, err := readTreeRoot(fsys, path)

	if err != nil {
		return LDSnap{}, err
	}

	return LDSnap{root: rNode, rPath: path}, nil
}

func (l LDSnap) FindNode(name string) (*LDSnapNode, error) {
	return l.findNode(name, l.rPath, l.root)
}

func (l LDSnap) findNode(name string, p string, pNode *LDSnapNode) (*LDSnapNode, error) {
	pNext := path.Join(p, pNode.name)

	if pNext == name {
		return pNode, nil
	}

	if pNode.isDir {
		for _, n := range pNode.children {
			pNext := path.Join()
			if node, err := l.findNode(name, pNext, n); err == nil {
				return node, nil
			}
		}
	}

	return nil, fmt.Errorf("could not find node with name %s", name)
}

func readTreeRoot(fsys fs.ReadDirFS, root string) (*LDSnapNode, error) {
	r, err := fsys.Open(root)

	if err != nil {
		return nil, err
	}

	info, err := r.Stat()

	if err != nil {
		return nil, err
	}

	node := &LDSnapNode{
		name:     info.Name(),
		modTime:  info.ModTime(),
		isDir:    true,
		children: map[string]*LDSnapNode{},
	}

	if err = readTreeChildren(fsys, root, node); err != nil {
		return nil, err
	}

	return node, nil
}

func readTreeChildren(fsys fs.ReadDirFS, p string, pNode *LDSnapNode) error {
	dEnts, err := fsys.ReadDir(p)

	if err != nil {
		return err
	}

	for _, dEnt := range dEnts {
		info, err := dEnt.Info()

		if err != nil {
			return err
		}

		node := &LDSnapNode{
			name:     info.Name(),
			modTime:  info.ModTime(),
			isDir:    info.IsDir(),
			children: map[string]*LDSnapNode{},
		}

		if info.IsDir() {
			if err = readTreeChildren(fsys, path.Join(p, info.Name()), node); err != nil {
				return err
			}
		}

		pNode.children[info.Name()] = node
	}

	return nil
}

type LDDiffType int

const (
	CREATED LDDiffType = iota
	MODIFIED
	DELETED
)

// node points to the node of the created or modified node entry in the new Snapshot
// unless dType is DELETED in which case it points to the node of the old Snapshot
type LDDiff struct {
	node  *LDSnapNode
	dType LDDiffType
}

func DiffSnaps(s1, s2 LDSnap, diffs chan LDDiff) {
	diffDirNodes(s1.root, s2.root, diffs)
	close(diffs)
}

func diffDirNodes(n1, n2 *LDSnapNode, diffs chan LDDiff) {
	// if n1 is nil then this is a new directory so add it and it's children
	if n1 == nil {
		for _, c := range n2.children {
			diffs <- LDDiff{c, CREATED}
			if c.isDir {
				diffDirNodes(nil, c, diffs)
			}
		}

		return
	}

	// check if any files have been deleted
	for name, c1 := range n1.children {
		if _, ok := n2.children[name]; !ok {
			diffs <- LDDiff{c1, DELETED}
		}
	}

	for name, c2 := range n2.children {
		c1, ok := n1.children[name]

		if ok && c1.modTime != c2.modTime {
			diffs <- LDDiff{c2, MODIFIED}
		} else if !ok {
			diffs <- LDDiff{c2, CREATED}
		}

		if c2.isDir && (c1.isDir || c1 == nil) {
			diffDirNodes(c1, c2, diffs)
		}
	}
}
