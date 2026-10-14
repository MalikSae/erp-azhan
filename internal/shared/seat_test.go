package shared

import "testing"

func TestCapSeatRestore(t *testing.T) {
	for _, tc := range []struct {
		name                string
		total, sisa, n      int
		want                int
		wantCapped          bool
	}{
		{"pengembalian biasa", 45, 10, 2, 12, false},
		{"tepat penuh", 45, 43, 2, 45, false},
		{"melebihi kapasitas terpotong", 45, 44, 2, 45, true},
		{"sudah penuh", 45, 45, 1, 45, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, capped := capSeatRestore(tc.total, tc.sisa, tc.n)
			if got != tc.want || capped != tc.wantCapped {
				t.Fatalf("capSeatRestore(%d,%d,%d)=%d,%v want %d,%v", tc.total, tc.sisa, tc.n, got, capped, tc.want, tc.wantCapped)
			}
		})
	}
}

func TestScheduleClosedReason(t *testing.T) {
	const today = "2026-10-08"
	for _, tc := range []struct {
		name, status, berangkat string
		closed                  bool
	}{
		{"published di masa depan", "published", "2026-11-01", false},
		{"draft di masa depan (pra-pendaftaran)", "draft", "2026-11-01", false},
		{"berangkat hari ini masih boleh", "published", "2026-10-08", false},
		{"diarsipkan", "archived", "2026-11-01", true},
		{"sudah berangkat", "published", "2026-10-07", true},
		{"draft yang tanggalnya lewat", "draft", "2026-01-01", true},
		{"tanggal kosong tidak dipakai menolak", "published", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ScheduleClosedReason(tc.status, tc.berangkat, today); (got != "") != tc.closed {
				t.Fatalf("ScheduleClosedReason(%q,%q)=%q closed=%v", tc.status, tc.berangkat, got, tc.closed)
			}
		})
	}
}
