package watcher

import (
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/sooraj-zebu/falcon/internal/storage"
)

type Event struct {
	File storage.FileInfo
	Type string
}

type Watcher struct {
	watcher *fsnotify.Watcher
	root    string
	Events  chan Event
	logger  *log.Logger
}

func New(root string, logger *log.Logger) (*Watcher, error) {

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		watcher: w,
		root:    root,
		Events:  make(chan Event, 100),
		logger:  logger,
	}, nil
}

func (w *Watcher) Start() error {

	// correctly walk filesystem
	err := filepath.Walk(w.root, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		// watch directories only
		if info.IsDir() {
			return w.watcher.Add(path)
		}

		return nil
	})

	if err != nil {
		return err
	}

	go w.loop()
	return nil
}

func (w *Watcher) loop() {

	for {
		select {

		case event := <-w.watcher.Events:

			w.Events <- Event{
				File: storage.FileInfo{
					Path:         event.Name,
					Name:         filepath.Base(event.Name),
					RelativePath: event.Name,
				},
				Type: event.Op.String(),
			}

		case err := <-w.watcher.Errors:
			w.logger.Println("watcher error:", err)
		}
	}
}

func (w *Watcher) Close() {
	_ = w.watcher.Close()
}
