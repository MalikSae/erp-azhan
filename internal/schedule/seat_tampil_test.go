package schedule

import (
	"context"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func intp(v int) *int { return &v }

// Scarcity seat: angka tampil = LEAST(seat_tampil, seat_sisa), nonaktif = seat_sisa.
func TestSeatTampilPublik(t *testing.T) {
	cases := []struct {
		tampil *int
		sisa   int
		want   int
	}{
		{nil, 45, 45},
		{intp(10), 45, 10},
		{intp(10), 4, 4},
		{intp(10), 10, 10},
		{intp(10), 0, 0},
	}
	for _, c := range cases {
		if got := SeatTampilPublik(c.tampil, c.sisa); got != c.want {
			t.Errorf("SeatTampilPublik(%v, %d) = %d, want %d", c.tampil, c.sisa, got, c.want)
		}
	}
}

// Scarcity aktif menjadikan paket buka berlabel Hampir penuh; SOLD_OUT dan CLOSED tetap dari seat ril.
func TestStatusRegistrasiPublikScarcity(t *testing.T) {
	batas := "2026-11-19"
	pkg := func(sisa int, tampil *int, berangkat string) *PublicSchedule {
		s := &Schedule{SeatTotal: 45, SeatSisa: sisa, SeatTampil: tampil, BerangkatTanggal: berangkat}
		return s.ToPublic()
	}
	cases := []struct {
		name string
		p    *PublicSchedule
		want string
	}{
		{"nonaktif, buka", pkg(45, nil, "2027-01-01"), RegOpen},
		{"aktif 10 dari 45", pkg(45, intp(10), "2027-01-01"), RegAlmostFull},
		{"aktif 30 dari 45", pkg(45, intp(30), "2027-01-01"), RegAlmostFull},
		{"angka >= sisa ril = nonaktif", pkg(20, intp(30), "2027-01-01"), RegOpen},
		{"habis ril", pkg(0, intp(10), "2027-01-01"), RegSoldOut},
		{"tutup tanggal", pkg(45, intp(10), "2026-11-01"), RegClosed},
	}
	for _, c := range cases {
		if got := statusRegistrasiPublik(c.p, batas); got != c.want {
			t.Errorf("%s: status = %s, want %s", c.name, got, c.want)
		}
	}
	if p := pkg(45, intp(10), "2027-01-01"); p.SeatSisa != 45 || p.SeatTampil != 10 {
		t.Errorf("ToPublic: seat_sisa=%d seat_tampil=%d, want 45 dan 10", p.SeatSisa, p.SeatTampil)
	}
}

// Ubah angka tampil: seat_sisa tidak berubah, riwayat jenis 'tampil' tercatat (0 = nonaktif).
func TestUpdateSeatTampilTx(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, _ := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	var actor int64
	if err := tx.QueryRow(`SELECT id FROM admin_users ORDER BY id LIMIT 1`).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	var sisaAwal, total int
	if err := tx.QueryRow(`SELECT seat_sisa, seat_total FROM schedules WHERE id=?`, sched).Scan(&sisaAwal, &total); err != nil {
		t.Fatal(err)
	}

	if err := updateSeatTampilTx(ctx, tx, sched, intp(10), nil, actor); err != nil {
		t.Fatalf("isi 10: %v", err)
	}
	if err := updateSeatTampilTx(ctx, tx, sched, intp(10), nil, actor); err != nil {
		t.Fatalf("isi ulang nilai sama: %v", err)
	}
	if err := updateSeatTampilTx(ctx, tx, sched, nil, nil, actor); err != nil {
		t.Fatalf("matikan: %v", err)
	}
	if err := updateSeatTampilTx(ctx, tx, sched, intp(total+1), nil, actor); !errors.Is(err, ErrConflict) {
		t.Fatalf("melebihi kuota: err=%v, want ErrConflict", err)
	}
	other := int64(-99)
	if err := updateSeatTampilTx(ctx, tx, sched, intp(5), &other, actor); !errors.Is(err, ErrNotFound) {
		t.Fatalf("brand lain: err=%v, want ErrNotFound", err)
	}

	var sisa int
	var tampil *int
	if err := tx.QueryRow(`SELECT seat_sisa, seat_tampil FROM schedules WHERE id=?`, sched).Scan(&sisa, &tampil); err != nil {
		t.Fatal(err)
	}
	if sisa != sisaAwal || tampil != nil {
		t.Fatalf("setelah dimatikan: seat_sisa=%d seat_tampil=%v, want %d dan NULL", sisa, tampil, sisaAwal)
	}

	rows, err := tx.Query(`SELECT jenis, admin_user_id, seat_before, seat_after FROM schedule_seat_adjustments WHERE schedule_id=? ORDER BY id`, sched)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	type catatan struct {
		jenis         string
		actor         int64
		before, after int
	}
	var got []catatan
	for rows.Next() {
		var c catatan
		if err := rows.Scan(&c.jenis, &c.actor, &c.before, &c.after); err != nil {
			t.Fatal(err)
		}
		got = append(got, c)
	}
	want := []catatan{{"tampil", actor, 0, 10}, {"tampil", actor, 10, 0}}
	if len(got) != len(want) {
		t.Fatalf("riwayat = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("riwayat[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}
