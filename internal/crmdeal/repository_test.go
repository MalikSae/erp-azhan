package crmdeal

import "testing"

func TestPhoneVariants(t *testing.T) {
	tests := []struct {
		input     string
		canonical string
		local     string
	}{
		{"+62 812-3456-7890", "6281234567890", "081234567890"},
		{"0812 3456 7890", "6281234567890", "081234567890"},
	}
	for _, test := range tests {
		canonical, local := phoneVariants(test.input)
		if canonical != test.canonical || local != test.local {
			t.Fatalf("phoneVariants(%q) = %q, %q", test.input, canonical, local)
		}
	}
}
