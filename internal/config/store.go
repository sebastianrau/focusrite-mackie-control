package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/sebastianrau/focusrite-mackie-control/internal/logging"
)

var log = logging.New("config")

// DefaultPath is the settings file in the user's config directory. It is the
// same location earlier releases used.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Monitor-Controller", "monitor-controller.yaml"), nil
}

// Store holds the settings and writes them back when they change. It is safe
// for concurrent use.
type Store struct {
	path string

	mu    sync.Mutex
	cfg   Config
	dirty bool
}

// Open loads the settings at path. A missing file yields defaults. An
// unreadable or invalid file is moved aside to path+".broken" and replaced
// by defaults, so a bad edit never keeps the app from starting.
func Open(path string) (*Store, error) {
	s := &Store{path: path}
	buf, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		log.Info("no settings file, using defaults", "path", path)
		s.cfg, s.dirty = Default(), true
		return s, nil
	case err != nil:
		return nil, err
	}

	cfg, migrated, err := parse(buf)
	if err != nil {
		broken := path + ".broken"
		log.Error("invalid settings file, using defaults", "path", path, "err", err, "moved_to", broken)
		if err := os.Rename(path, broken); err != nil {
			return nil, fmt.Errorf("move invalid settings aside: %w", err)
		}
		s.cfg, s.dirty = Default(), true
		return s, nil
	}
	if migrated {
		log.Info("migrated settings from an earlier release", "path", path)
	}
	s.cfg, s.dirty = cfg, migrated
	return s, nil
}

func parse(buf []byte) (cfg Config, migrated bool, err error) {
	var probe struct {
		Version int `yaml:"version"`
	}
	if err := yaml.Unmarshal(buf, &probe); err != nil {
		return Config{}, false, err
	}
	switch probe.Version {
	case 0:
		var l legacyConfig
		if err := yaml.Unmarshal(buf, &l); err != nil {
			return Config{}, false, err
		}
		return l.migrate(), true, nil
	case Version:
		cfg = Default()
		dec := yaml.NewDecoder(bytes.NewReader(buf))
		dec.KnownFields(true)
		if err := dec.Decode(&cfg); err != nil {
			return Config{}, false, err
		}
		return cfg, false, nil
	default:
		return Config{}, false, fmt.Errorf("unsupported settings version %d", probe.Version)
	}
}

func (s *Store) Path() string { return s.path }

// Get returns a copy of the settings.
func (s *Store) Get() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return clone(s.cfg)
}

// Update changes the settings through fn and marks them for saving if
// anything changed.
func (s *Store) Update(fn func(*Config)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := clone(s.cfg)
	fn(&next)
	if !reflect.DeepEqual(next, s.cfg) {
		s.cfg, s.dirty = next, true
	}
}

// Save writes the settings if they changed since the last save. The file is
// replaced atomically, so a crash never leaves a truncated file behind.
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.dirty {
		return nil
	}
	buf, err := yaml.Marshal(s.cfg)
	if err != nil {
		return err
	}
	if err := writeAtomic(s.path, buf); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	s.dirty = false
	log.Debug("settings saved", "path", s.path)
	return nil
}

// AutoSave saves changes every interval until ctx is cancelled.
func (s *Store) AutoSave(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := s.Save(); err != nil {
				log.Error("autosave failed", "err", err)
			}
		}
	}
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name()) // no-op after a successful rename
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func clone(c Config) Config {
	c.Controller.Speakers = append([]Speaker(nil), c.Controller.Speakers...)
	c.Focusrite.Outputs = append([]Output(nil), c.Focusrite.Outputs...)
	c.MIDI.SpeakerKeys = append([]Key(nil), c.MIDI.SpeakerKeys...)
	return c
}
