package schedule

import "testing"

func TestRegistrationStatus(t *testing.T) {
	const batas = "2026-11-19"
	cases := []struct {
		nama        string
		total, sisa int
		berangkat   string
		want        string
	}{
		{"seat lega", 45, 25, "2026-12-20", RegOpen},
		{"tepat di atas ambang 20% (45 seat -> ambang 9)", 45, 10, "2026-12-20", RegOpen},
		{"tepat di ambang 20% (45 seat -> ambang 9)", 45, 9, "2026-12-20", RegAlmostFull},
		{"kuota kecil memakai minimal 5 seat", 10, 5, "2026-12-20", RegAlmostFull},
		{"kuota kecil, di atas 5 seat", 10, 6, "2026-12-20", RegOpen},
		{"kuota besar, 20% di atas 5 seat", 100, 20, "2026-12-20", RegAlmostFull},
		{"kuota besar, di atas 20%", 100, 21, "2026-12-20", RegOpen},
		{"sisa 1 seat", 45, 1, "2026-12-20", RegAlmostFull},
		{"sisa 0 = sold out", 45, 0, "2026-12-20", RegSoldOut},
		{"sisa negatif = sold out", 45, -2, "2026-12-20", RegSoldOut},
		{"berangkat tepat di batas masih dibuka", 45, 25, batas, RegOpen},
		{"berangkat sebelum batas = ditutup", 45, 25, "2026-11-18", RegClosed},
		{"sold out lebih diutamakan daripada ditutup", 45, 0, "2026-11-18", RegSoldOut},
		{"ditutup lebih diutamakan daripada hampir penuh", 45, 3, "2026-11-18", RegClosed},
	}
	for _, c := range cases {
		if got := registrationStatus(c.total, c.sisa, c.berangkat, batas); got != c.want {
			t.Errorf("%s: registrationStatus(%d, %d, %s) = %s, want %s", c.nama, c.total, c.sisa, c.berangkat, got, c.want)
		}
	}
}

func TestDurationDays(t *testing.T) {
	cases := []struct {
		berangkat, pulang string
		want              int
	}{
		{"2026-12-20", "2026-12-28", 9},
		{"2026-12-20", "2026-12-31", 12},
		{"2026-12-28", "2027-01-05", 9}, // melewati pergantian tahun
		{"2026-12-20", "2026-12-20", 1},
		{"2026-12-20", "2026-12-19", 0}, // pulang sebelum berangkat
		{"", "2026-12-20", 0},
		{"2026-12-20", "bukan-tanggal", 0},
	}
	for _, c := range cases {
		if got := durationDays(c.berangkat, c.pulang); got != c.want {
			t.Errorf("durationDays(%q, %q) = %d, want %d", c.berangkat, c.pulang, got, c.want)
		}
	}
}
