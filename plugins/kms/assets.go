package kms_plugin_assets

import (
	"io/fs"
	"path"
	"runtime"
)

const contentDir = "assets"

func FileSystem() fs.FS {
	// Remove the root
	dir := path.Join(contentDir, runtime.GOOS, runtime.GOARCH)
	f, err := fs.Sub(content, dir)
	if err != nil {
		panic(err)
	}
	return f
}
