package usecase

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/asd2003ru/git-safe/internal/domain"
)

func Test_fullPath(t *testing.T) {
	want, err := filepath.Abs("service_test.go")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
	}{
		{
			name: "absolute path",
			path: "service_test.go",
		},
		{
			name: "relative path",
			path: "./service_test.go",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fullPath(tt.path)
			if got != want {
				t.Errorf("fullPath() = %v, want %v", got, want)
			}
		})
	}
}

func TestDirectoryPathHelpers(t *testing.T) {
	list := domain.FileList{
		Version: 7,
		Files: []domain.SecureFile{
			{Path: "secrets/root"},
			{Path: "secrets/nested/child"},
			{Path: "secrets-other/file"},
			{Path: "public"},
		},
		Directories: []domain.SecureDirectory{
			{Path: "secrets"},
			{Path: "public"},
		},
	}

	files := filesInDirectory(list.Files, "./secrets")
	if !reflect.DeepEqual(files, []domain.SecureFile{{Path: "secrets/root"}, {Path: "secrets/nested/child"}}) {
		t.Fatalf("unexpected files in directory: %#v", files)
	}

	updated := removeDirectoryFromList(list, "secrets")
	if updated.Version != list.Version {
		t.Fatalf("version changed: %d", updated.Version)
	}
	if !reflect.DeepEqual(updated.Directories, []domain.SecureDirectory{{Path: "public"}}) {
		t.Fatalf("unexpected directories after remove: %#v", updated.Directories)
	}
	if !reflect.DeepEqual(updated.Files, []domain.SecureFile{{Path: "secrets-other/file"}, {Path: "public"}}) {
		t.Fatalf("unexpected files after remove: %#v", updated.Files)
	}
}

func TestPathInDirectory(t *testing.T) {
	tests := []struct {
		name string
		path string
		dir  string
		want bool
	}{
		{name: "direct file", path: "secrets/root", dir: "secrets", want: true},
		{name: "nested file", path: "secrets/nested/child", dir: "./secrets", want: true},
		{name: "directory itself", path: "secrets", dir: "secrets", want: true},
		{name: "sibling prefix", path: "secrets-other/file", dir: "secrets", want: false},
		{name: "repository root excludes dot", path: ".", dir: ".", want: false},
		{name: "repository root includes child", path: "file", dir: ".", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pathInDirectory(tt.path, tt.dir); got != tt.want {
				t.Fatalf("pathInDirectory(%q, %q) = %v, want %v", tt.path, tt.dir, got, tt.want)
			}
		})
	}
}
