package main

import ()

type Watcher interface {
	Events() chan Event
	Errors() chan error
}

type Client struct {
	watcher Watcher
}
