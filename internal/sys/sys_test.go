package sys

import (
	"os"
	"testing"
)

func TestIsWayland(t *testing.T) {
	tests := []struct {
		name           string
		waylandDisplay string
		sessionType    string
		want           bool
	}{
		{
			name:           "wayland display set",
			waylandDisplay: "wayland-0",
			sessionType:    "",
			want:           true,
		},
		{
			name:           "session type wayland",
			waylandDisplay: "",
			sessionType:    "wayland",
			want:           true,
		},
		{
			name:           "both set",
			waylandDisplay: "wayland-0",
			sessionType:    "wayland",
			want:           true,
		},
		{
			name:           "x11 session",
			waylandDisplay: "",
			sessionType:    "x11",
			want:           false,
		},
		{
			name:           "nothing set",
			waylandDisplay: "",
			sessionType:    "",
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origWayland := os.Getenv("WAYLAND_DISPLAY")
			origSession := os.Getenv("XDG_SESSION_TYPE")
			defer func() {
				os.Setenv("WAYLAND_DISPLAY", origWayland)
				os.Setenv("XDG_SESSION_TYPE", origSession)
			}()

			os.Setenv("WAYLAND_DISPLAY", tt.waylandDisplay)
			os.Setenv("XDG_SESSION_TYPE", tt.sessionType)

			if got := IsWayland(); got != tt.want {
				t.Errorf("IsWayland() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRuntimeDir(t *testing.T) {
	tests := []struct {
		name       string
		runtimeDir string
		want       string
	}{
		{
			name:       "xdg runtime dir set",
			runtimeDir: "/run/user/1000",
			want:       "/run/user/1000",
		},
		{
			name:       "xdg runtime dir empty",
			runtimeDir: "",
			want:       "/tmp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")
			defer os.Setenv("XDG_RUNTIME_DIR", origRuntimeDir)

			os.Setenv("XDG_RUNTIME_DIR", tt.runtimeDir)

			if got := GetRuntimeDir(); got != tt.want {
				t.Errorf("GetRuntimeDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	s := New(nil)
	if s == nil {
		t.Error("New() should not return nil")
	}
	if s.conn != nil {
		t.Error("conn should be nil when passed nil")
	}
}
