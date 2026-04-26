package web

import (
	"io/fs"
	"testing"
)

func TestFiles(t *testing.T) {
	files, err := Files()
	if err != nil {
		t.Fatalf("Files() returned error: %v", err)
	}

	for _, name := range []string{"index.html", "app.js", "style.css"} {
		if _, err := fs.Stat(files, name); err != nil {
			t.Fatalf("static file %q is not available: %v", name, err)
		}
	}
}
