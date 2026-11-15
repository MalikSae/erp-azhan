package customer

import "testing"

func TestSusunRute(t *testing.T) {
	cases := []struct{ asal, tujuan, want string }{
		{"CGK", "JED", "CGK → JED"},
		{" CGK ", "MED", "CGK → MED"},
		{"CGK", "", ""},
		{"", "JED", ""},
	}
	for _, c := range cases {
		if got := susunRute(c.asal, c.tujuan); got != c.want {
			t.Errorf("susunRute(%q, %q) = %q, mau %q", c.asal, c.tujuan, got, c.want)
		}
	}
}
