// internal/provider/provider_test.go
package provider

import (
	"testing"
)

func TestSearchOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    SearchOptions
		wantErr bool
	}{
		{
			name:    "valid options",
			opts:    SearchOptions{Mode: "ask", MaxTokens: 1024},
			wantErr: false,
		},
		{
			name:    "empty mode is valid (uses profile default)",
			opts:    SearchOptions{},
			wantErr: false,
		},
		{
			name:    "negative max tokens",
			opts:    SearchOptions{MaxTokens: -1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResultHasSources(t *testing.T) {
	r := &Result{
		Content: "answer text",
		Sources: []Source{{Title: "src", URL: "https://example.com"}},
	}
	if len(r.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(r.Sources))
	}

	empty := &Result{Content: "no sources"}
	if len(empty.Sources) != 0 {
		t.Errorf("expected 0 sources, got %d", len(empty.Sources))
	}
}
