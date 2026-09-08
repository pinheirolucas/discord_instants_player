package instant

import "testing"

func TestIsLinkValid(t *testing.T) {
	tests := []struct {
		name string
		link string
		want bool
	}{
		{"absolute https url", "https://www.myinstants.com/media/sounds/a.mp3", true},
		{"absolute http url", "http://localhost:9001/clip.mp3", true},
		{"url with query string", "https://example.com/a.mp3?x=1", true},
		{"non-http scheme still parses as absolute", "ftp://example.com/a.mp3", true},
		{"empty string", "", false},
		{"absolute path without scheme", "/media/sounds/a.mp3", false},
		{"host without scheme", "myinstants.com/a.mp3", false},
		{"scheme without host", "http://", false},
		{"not a url at all", "not a url", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLinkValid(tt.link); got != tt.want {
				t.Errorf("IsLinkValid(%q) = %v, want %v", tt.link, got, tt.want)
			}
		})
	}
}
