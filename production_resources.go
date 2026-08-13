//go:build !bindings

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

func validateProductionResources(playerAssets fs.FS, bundledDemo string) error {
	if playerAssets == nil {
		return errors.New("player assets are unavailable")
	}
	if info, err := fs.Stat(playerAssets, "index.html"); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("player assets are incomplete: index.html is unavailable")
	}
	if info, err := os.Stat(bundledDemo); err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("bundled demo is unavailable at %s", bundledDemo)
	}
	return nil
}
