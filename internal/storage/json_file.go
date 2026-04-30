package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type JSONFile[T any] struct {
	mu        sync.Mutex
	path      string
	data      T
	normalize func(*T)
}

func OpenJSONFile[T any](path string, initial T, normalize func(*T)) (*JSONFile[T], error) {
	f := &JSONFile[T]{
		path:      path,
		data:      initial,
		normalize: normalize,
	}
	if normalize != nil {
		normalize(&f.data)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := f.saveLocked(); err != nil {
				return nil, err
			}
			return f, nil
		}
		return nil, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &f.data); err != nil {
			return nil, err
		}
	}
	if normalize != nil {
		normalize(&f.data)
	}
	return f, nil
}

func (f *JSONFile[T]) View(fn func(T) error) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, err := clone(f.data)
	if err != nil {
		return err
	}
	return fn(data)
}

func (f *JSONFile[T]) Update(fn func(*T) error) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.normalize != nil {
		f.normalize(&f.data)
	}
	if err := fn(&f.data); err != nil {
		return err
	}
	if f.normalize != nil {
		f.normalize(&f.data)
	}
	return f.saveLocked()
}

func (f *JSONFile[T]) Snapshot() (T, error) {
	var out T
	err := f.View(func(data T) error {
		out = data
		return nil
	})
	return out, err
}

func (f *JSONFile[T]) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(f.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}

func clone[T any](v T) (T, error) {
	var out T
	raw, err := json.Marshal(v)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(raw, &out)
	return out, err
}
