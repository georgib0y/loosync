package main

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
)

type Client struct {
	watcher fsnotify.Watcher
}
