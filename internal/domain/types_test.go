package domain

import "testing"

func TestStatusCodeString(t *testing.T) {
	tests := []struct {
		code StatusCode
		want string
	}{
		{code: NotHidden, want: "not hidden"},
		{code: HiddenInSync, want: "hidden, in sync"},
		{code: HiddenModified, want: "hidden, modified"},
		{code: HiddenNotRevealed, want: "hidden, not revealed"},
		{code: HiddenPrivateMissing, want: "WARNING: private file missing!"},
		{code: StatusCode(0), want: "unknown"},
	}

	for _, tt := range tests {
		if got := tt.code.String(); got != tt.want {
			t.Fatalf("StatusCode(%d).String() = %q, want %q", tt.code, got, tt.want)
		}
	}
}
