package main

import (
	"encoding/json"
	"io"
	"os"
)

type EventType int

const (
	E_CREATED EventType = iota
	E_MODIFIED
	E_DELETED
)

type Event struct {
	path  string
	eType EventType
}

type DB interface {
	PushEvent(e Event) error
	PopAllEvents() []Event
}

type JsonDB struct {
	path   string
	events map[string]Event
}

func NewJsonDB(path string) (JsonDB, error) {
	file, err := os.Open(path)
	defer file.Close()

	var events map[string]Event
	switch err {
	case nil:
		events, err = readEventsFromReader(file)
		if err != nil {
			return JsonDB{}, err
		}
	case os.ErrNotExist:
		event = map[string]Event{}
	default:
		return JsonDB{}, err
	}

	return JsonDB{path, events}, nil
}

func readEventsFromReader(r io.Reader) (map[string]Event, error) {
	dec := json.NewDecoder(r)

	var events map[string]Event
	if err := dec.Decode(&events); err != nil {
		return nil, err
	}

	return events, nil
}
