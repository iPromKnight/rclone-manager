package watcher

import (
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
)

type Watcher struct {
	watcher  *fsnotify.Watcher
	callback func(string)
	logger   zerolog.Logger
}

func NewWatcher(callback func(string), logger zerolog.Logger) *Watcher {
	w, _ := fsnotify.NewWatcher()
	return &Watcher{watcher: w, callback: callback, logger: logger}
}

func (w *Watcher) Watch(files []string) {
	for _, file := range files {
		_ = w.watcher.Add(file)
		w.logger.Info().Str("file", file).Msg("Watching for changes...")
	}

	go func() {
		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				if event.Op == fsnotify.Write || event.Op == fsnotify.Create {
					w.logger.Info().Str("file", event.Name).Msg("File modified, triggering reload")
					w.callback(event.Name)
				}
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				w.logger.Error().Err(err).Msg("Error watching file")
			}
		}
	}()
}

func (w *Watcher) Close() {
	_ = w.watcher.Close()
}
