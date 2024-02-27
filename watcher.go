package main

import (
	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	w       *fsnotify.Watcher
	changes ChangeRepo
}

func NewWatcher(root string, changeRepo ChangeRepo) (Watcher, error) {
	return Watcher{}, nil
}
