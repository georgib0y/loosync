package main

import (
	"fmt"
	"io/fs"
	"log"
	"path"
	"time"
)

const POLLER_INTERVAL time.Duration = 1 * time.Second

type EType int

func (e EType) String() string {
	switch e {
	case CREATED:
		return "CREATED"
	case MODIFIED:
		return "MODIFIED"
	case DELETED:
		return "DELETED"
	default:
		return "UNKNOWN"
	}
}

const (
	CREATED EType = iota
	MODIFIED
	DELETED
)

type Event struct {
	name  string
	eType EType
}

func (e Event) String() string {
	return fmt.Sprintf("NAME: %s, EVENT_TYPE: %s", e.name, e.eType)
}

type node struct {
	name     string
	isDir    bool
	modTime  time.Time
	children map[string]*node
}

func newNodeFromInfo(info fs.FileInfo) *node {
	return &node{
		name:     info.Name(),
		modTime:  info.ModTime(),
		isDir:    info.IsDir(),
		children: map[string]*node{},
	}
}

type Poller struct {
	fsys   fs.ReadDirFS
	root   string
	node   *node
	events chan Event
	errors chan error
}

func NewPoller(fsys fs.ReadDirFS, root string) (*Poller, error) {
	// check if root is not a directory
	rDir, err := fsys.Open(root)
	if err != nil {
		return nil, err
	}

	info, err := rDir.Stat()
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("root path is not a directory: %s", root)
	}

	events := make(chan Event)
	errors := make(chan error)

	return &Poller{fsys, root, nil, events, errors}, nil
}

func (p *Poller) Close() error {
	close(p.events)
	close(p.errors)

	return nil
}

func (p *Poller) diff() {
	// if node is nill then populate node and return (no changes will be found)
	if p.node == nil {
		p.node = p.nextSnap()
		return
	}

	next := p.nextSnap()
	p.diffNodes(p.node, next, p.root)

	// set the new node as the current node for future diffing
	p.node = next
}

func (p *Poller) diffNodes(orig, next *node, currPath string) {
	// if orig is nil then next is a new dir so add its children
	if orig == nil {
		for _, c := range next.children {
			p.events <- Event{path.Join(currPath, c.name), CREATED}
			if c.isDir {
				p.diffNodes(nil, c, path.Join(currPath, c.name))
			}
		}

		return
	}

	// check if any files have been deleted
	// for name, c := range orig.children {
	for name, c := range orig.children {
		if _, ok := next.children[name]; !ok {
			p.events <- Event{path.Join(currPath, c.name), DELETED}
		}
	}

	for nextName, cNext := range next.children {
		cOrig, exists := orig.children[nextName]

		// check if any files have been modified since
		if exists && cOrig.modTime.Before(cNext.modTime) {
			p.events <- Event{path.Join(currPath, cNext.name), MODIFIED}
		}

		// check that the child exists in orig, if not then a new file has been made
		if !exists {
			p.events <- Event{path.Join(currPath, cNext.name), CREATED}
		}

		if cNext.isDir && (cOrig.isDir || cOrig == nil) {
			log.Println("Entering into subdir: ", path.Join(currPath, cNext.name))
			p.diffNodes(cOrig, cNext, path.Join(currPath, cNext.name))
		}
	}
}

func (p *Poller) nextSnap() *node {
	rDir, err := p.fsys.Open(p.root)
	if err != nil {
		p.errors <- err
		return nil
	}

	info, err := rDir.Stat()
	if err != nil {
		p.errors <- err
		return nil
	}

	n := newNodeFromInfo(info)
	p.readNodeChildren(p.root, n)
	return n
}

func (p *Poller) readNodeChildren(name string, parent *node) {
	entries, err := p.fsys.ReadDir(name)
	if err != nil {
		p.errors <- err
		return
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			p.errors <- err
			continue
		}

		n := newNodeFromInfo(info)
		if n.isDir {
			p.readNodeChildren(path.Join(name, n.name), n)
		}

		parent.children[n.name] = n
	}
}
