package cmd

import (
	"testing"

	"github.com/asd2003ru/git-safe/internal/domain"
)

func TestStatusHint(t *testing.T) {
	tests := []struct {
		name   string
		status domain.FileStatus
		want   string
	}{
		{
			name: "missing private file",
			status: domain.FileStatus{
				File:   domain.SecureFile{Path: "secret"},
				Status: domain.HiddenPrivateMissing,
			},
			want: "fix: run `git-safe hide secret` if plaintext is current, or restore secret.safe from git/backup",
		},
		{
			name: "missing source and private files",
			status: domain.FileStatus{
				File:   domain.SecureFile{Path: "secret"},
				Status: domain.HiddenMissing,
			},
			want: "fix: restore secret.safe from git/backup, or run `git-safe remove secret` to stop tracking it",
		},
		{
			name: "in sync",
			status: domain.FileStatus{
				File:   domain.SecureFile{Path: "secret"},
				Status: domain.HiddenInSync,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusHint(tt.status); got != tt.want {
				t.Fatalf("statusHint() = %q, want %q", got, tt.want)
			}
		})
	}
}
