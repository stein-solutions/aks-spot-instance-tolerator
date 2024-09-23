package util

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type SecretWatcher struct {
	loadMu              sync.Mutex
	watchMu             sync.Mutex
	secretData          map[string]string
	secretDataOverrides map[string]string
	path                string
}

func NewSecretWatcher(path string) *SecretWatcher {
	sw := &SecretWatcher{
		loadMu:              sync.Mutex{},
		watchMu:             sync.Mutex{},
		secretData:          make(map[string]string),
		secretDataOverrides: make(map[string]string),
		path:                path,
	}

	return sw
}

func (sw *SecretWatcher) PutSecretDataOverride(key, value string) {
	sw.loadMu.Lock()
	defer sw.loadMu.Unlock()
	sw.secretDataOverrides[key] = value
}

func (sw *SecretWatcher) GetSecretData(key string) (string, bool) {
	sw.loadMu.Lock()
	defer sw.loadMu.Unlock()

	if value, exist := sw.secretDataOverrides[key]; exist {
		return value, true
	}
	value, exists := sw.secretData[key]
	return value, exists
}

func (sw *SecretWatcher) LoadSecret() error {
	sw.loadMu.Lock()
	defer sw.loadMu.Unlock()
	slog.Debug(fmt.Sprintf("Loading secret - From path: %s", sw.path))

	sw.secretDataOverrides = make(map[string]string) // reset overrides

	fileInfo, err := os.Stat(sw.path)
	if err != nil {
		return err
	}

	filesToLoad := []string{}

	if fileInfo.IsDir() {
		dir, err := os.ReadDir(sw.path)
		if err != nil {
			return err
		}

		slog.Debug(fmt.Sprintf("Loading secret - Found: %s files", strconv.Itoa(len(dir))))
		for _, file := range dir {
			filesToLoad = append(filesToLoad, filepath.Join(sw.path, file.Name()))
		}
	} else {
		filesToLoad = append(filesToLoad, sw.path)
	}

	fileData := make(map[string]string)
	for _, fileToLoad := range filesToLoad {
		fileInfo, err := os.Stat(fileToLoad)
		if err != nil {
			return err
		}
		if fileInfo.IsDir() {
			continue
		}
		slog.Debug(fmt.Sprintf("Loading secret - Loading File: %s", fileToLoad))
		secretData, err := os.ReadFile(fileToLoad)
		if err != nil {
			return err
		}

		fileData[fileToLoad] = string(secretData)
	}

	sw.secretData = fileData

	return nil
}

func (sw *SecretWatcher) WatchSecret() error {
	sw.watchMu.Lock()
	defer sw.watchMu.Unlock()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	err = watcher.Add(sw.path)
	if err != nil {
		return err
	}

	go func() error {
		slog.Debug("WatchSecret - Starting Watcher.")
		for {
			slog.Debug("WatchSecret - In Wacher Loop.")

			select {
			case event, ok := <-watcher.Events:
				slog.Debug(fmt.Sprintf("WatchSecret - Got Event %s (%s) with status %s.", event.Name, event.Op.String(), strconv.FormatBool(ok)))
				if !ok {
					continue
				}

				if event.Has(fsnotify.Write | fsnotify.Create) {
					slog.Debug(fmt.Sprintf("WatchSecret - Write event fired for file: %s", event.Name))
					if err := sw.LoadSecret(); err != nil {
						slog.Error(fmt.Sprintf("WatchSecret - Got error event %s", err))
						return fmt.Errorf("error loading secret: %v", err)
					}
				}

			case err := <-watcher.Errors:
				if err != nil {
					slog.Error(fmt.Sprintf("WatchSecret - Got error event %s", err))
					return fmt.Errorf("error watching file: %v", err)
				}
			}
		}
	}()

	err = sw.LoadSecret()
	if err != nil {
		return err
	}

	return nil
}
