package agen

import "testing"

func TestKodeAgenValid(t *testing.T) {
	cases := []struct {
		kode string
		want bool
	}{
		{"ABC234", true},
		{"abc234", true},
		{" abc234 ", true},
		{"ABCDE", false},
		{"ABC2345", false},
		{"ABC-23", false},
		{"", false},
	}
	for _, c := range cases {
		if got := KodeAgenValid(c.kode); got != c.want {
			t.Errorf("KodeAgenValid(%q) = %v, want %v", c.kode, got, c.want)
		}
	}
}
