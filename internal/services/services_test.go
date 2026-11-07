package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bambuo/rclick-go/internal/config"
)

func TestResolveTemplate(t *testing.T) {
	got := ResolveTemplate(`chmod +x "{path}"`, []string{"/tmp/a b.txt"})
	if want := `chmod +x "/tmp/a b.txt"`; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	got = ResolveTemplate(`zip -r "{name_no_ext}.zip" "{path}"`, []string{"/tmp/photo.jpg"})
	if want := `zip -r "photo.zip" "/tmp/photo.jpg"`; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	got = ResolveTemplate(`echo {all}`, []string{"/a", "/b"})
	if want := `echo "/a" "/b"`; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	got = ResolveTemplate(`ls {dir} {ext} {name}`, []string{"/tmp/report.pdf"})
	if want := `ls /tmp/report .pdf report.pdf`; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPlanNewFilePath(t *testing.T) {
	dir := t.TempDir()
	existing := map[string]bool{
		filepath.Join(dir, "Untitled.txt"):  true,
		filepath.Join(dir, "Untitled1.txt"): true,
	}
	exists := func(path string) bool { return existing[path] }

	got := PlanNewFilePath(dir, "txt", "Untitled", exists)
	if want := filepath.Join(dir, "Untitled2.txt"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	if got := NormalizeExtension(".md"); got != ".md" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeExtension("md"); got != ".md" {
		t.Fatalf("got %q", got)
	}
}

func TestProtectedPaths(t *testing.T) {
	home := "/Users/test"
	cases := map[string]bool{
		"/Users/test":           true,
		"/Users/test/Desktop":   true,
		"/Users/test/Documents": false,
		"/System/Library":       true,
		"/usr/bin":              true,
		"/Applications":         true,
		"/Users":                true,
		"/tmp/work/file.txt":    false,
	}
	for path, want := range cases {
		if got := IsProtected(path, home); got != want {
			t.Errorf("IsProtected(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestCreateFileUsesUniqueNames(t *testing.T) {
	dir := t.TempDir()
	store, err := config.Open(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	files := &FileOperations{Store: store, Home: dir}

	if err := files.CreateFile(dir, "newfile-txt"); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if err := files.CreateFile(dir, "newfile-txt"); err != nil {
		t.Fatalf("second create: %v", err)
	}

	for _, name := range []string{"Untitled.txt", "Untitled1.txt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestCopyPathsDecodesFirstPath(t *testing.T) {
	var copied string
	files := &FileOperations{
		Home:      t.TempDir(),
		Clipboard: func(text string) error { copied = text; return nil },
	}
	if err := files.CopyPaths([]string{"/tmp/a%20b.txt", "/tmp/other"}); err != nil {
		t.Fatal(err)
	}
	if copied != "/tmp/a b.txt" {
		t.Fatalf("copied %q", copied)
	}
}
