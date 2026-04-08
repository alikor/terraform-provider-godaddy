package client

import "testing"

func TestResolveBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		endpoint string
		override string
		want     string
	}{
		{
			name:     "production default",
			endpoint: "production",
			want:     "https://api.godaddy.com",
		},
		{
			name:     "ote default",
			endpoint: "ote",
			want:     "https://api.ote-godaddy.com",
		},
		{
			name:     "override wins",
			endpoint: "production",
			override: "https://example.test/",
			want:     "https://example.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ResolveBaseURL(tt.endpoint, tt.override)
			if got != tt.want {
				t.Fatalf("ResolveBaseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
