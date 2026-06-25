package versions

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/domain"
)

var ErrNotCached = errors.New("not cached")

type Cache struct {
	dir string
}

func NewCache(dir string) *Cache { return &Cache{dir: dir} }

func (c *Cache) path(t domain.ServerType, version string) string {
	return filepath.Join(c.dir, string(t), version+".jar")
}

func (c *Cache) Get(t domain.ServerType, version string) (string, error) {
	p := c.path(t, version)
	if _, err := os.Stat(p); err != nil {
		return "", ErrNotCached
	}
	return p, nil
}

func (c *Cache) Store(t domain.ServerType, version, srcPath string) error {
	dest := c.path(t, version)
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	return copyFile(srcPath, dest)
}

func (c *Cache) List(t domain.ServerType) ([]string, error) {
	dir := filepath.Join(c.dir, string(t))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			out = append(out, fmt.Sprintf("%s (cached)", e.Name()))
		}
	}
	return out, nil
}
