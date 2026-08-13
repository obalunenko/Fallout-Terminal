//go:build bindings

package main

import "io/fs"

// Wails generates bindings before it runs frontend:build. The temporary
// binding binary must therefore tolerate the checked-in placeholder assets;
// the final production binary is compiled after Wails builds both frontends.
func validateProductionResources(fs.FS, string) error {
	return nil
}
