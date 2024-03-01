package main

import (
	"github.com/fsnotify/fsnotify"
)

type Client struct {
	watcher fsnotify.Watcher
}
