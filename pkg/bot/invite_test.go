package bot

import "testing"

func TestParseInviteCode(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantOk  bool
	}{
		{"discord.gg link", "https://discord.gg/rqPBH6eC", "rqPBH6eC", true},
		{"discord.com/invite link", "https://discord.com/invite/rqPBH6eC", "rqPBH6eC", true},
		{"discordapp.com/invite link", "https://discordapp.com/invite/rqPBH6eC", "rqPBH6eC", true},
		{"link embedded in surrounding text", "join us: discord.gg/abc123 see you there", "abc123", true},
		{"plain chat message", "hello there", "", false},
		{"empty message", "", "", false},
		{"unrelated url", "https://myinstants.com/en/index/us/", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseInviteCode(tt.content)
			if ok != tt.wantOk {
				t.Fatalf("parseInviteCode(%q) ok = %v, want %v", tt.content, ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("parseInviteCode(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}
