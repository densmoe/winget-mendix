package main

import "testing"

func TestProductCodeFor(t *testing.T) {
	tests := []struct {
		version  string
		expected string
	}{
		{"10.18.0.54340", "{b271467405adbc1c2844d4fe816de008245b812c_is1}"},
		{"10.12.8.48735", "{27030f22c2ef96646e462cd22d9b5bf3db38f400_is1}"},
		{"10.12.4.44290", "{f1b124cafafabcef2fa45d133a5698a9e7e7b051_is1}"},
		{"11.9.1", "{2b036394de66dc3458280799e7de943d04dff8c0_is1}"},
	}

	for _, tt := range tests {
		got := ProductCodeFor(tt.version)
		if got != tt.expected {
			t.Errorf("ProductCodeFor(%q) = %q, want %q", tt.version, got, tt.expected)
		}
	}
}
