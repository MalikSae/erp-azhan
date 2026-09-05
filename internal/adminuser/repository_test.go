package adminuser

import "testing"

func TestDisplayNameFromEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{name: "normal email", email: "admin2@hana.id", want: "admin2"},
		{name: "trim whitespace", email: "  admin@hana.id  ", want: "admin"},
		{name: "missing local part", email: "@hana.id", want: "Admin"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := displayNameFromEmail(test.email); got != test.want {
				t.Fatalf("displayNameFromEmail(%q) = %q, want %q", test.email, got, test.want)
			}
		})
	}
}
