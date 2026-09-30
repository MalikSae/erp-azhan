package schedule

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"erp-azhan/api/internal/testdb"
	"github.com/go-chi/chi/v5"
)

func TestCapacityChangePreservesAllocatedSeats(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		total, remaining, next, want int
		invalid                      bool
	}{
		{"content edit after booking", 45, 44, 45, 44, false},
		{"increase capacity", 45, 35, 50, 40, false},
		{"reduce to allocation", 45, 35, 10, 0, false},
		{"below allocation", 45, 35, 9, 0, true},
		{"corrupt remaining", 45, 46, 45, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := remainingAfterCapacityChange(tc.total, tc.remaining, tc.next)
			if (err != nil) != tc.invalid || got != tc.want {
				t.Fatalf("got %d, %v; want %d invalid=%v", got, err, tc.want, tc.invalid)
			}
		})
	}
}

func TestMinimalDPContract(t *testing.T) {
	for _, tc := range []struct {
		value   float64
		invalid bool
	}{{-1, true}, {0, false}, {5_000_000, false}, {20_000_000, false}, {20_000_001, true}} {
		if err := validateDP(&tc.value, 20_000_000, 22_000_000, 25_000_000); (err != nil) != tc.invalid {
			t.Fatalf("DP %v: %v", tc.value, err)
		}
	}
	if err := validateDP(nil, 20_000_000); err != nil {
		t.Fatal(err)
	}
}

func TestPublicDetailRequiresBrand(t *testing.T) {
	router := chi.NewRouter()
	router.Get("/api/schedules/{id}", NewHandler(nil).GetSchedulePublic)
	for _, suffix := range []string{"", "?brand=0", "?brand=-1", "?brand=abc"} {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/schedules/43"+suffix, nil))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", suffix, rr.Code)
		}
	}
}

func TestPublicScheduleEffectiveDPProjection(t *testing.T) {
	_, tx := testdb.Tx(t)
	var id, brandID int64
	if err := tx.QueryRow(`SELECT id, brand_id FROM schedules ORDER BY id LIMIT 1`).Scan(&id, &brandID); err != nil {
		t.Fatal(err)
	}
	testdb.Exec(t, tx, `UPDATE brands SET minimal_dp=7000000 WHERE id=?`, brandID)
	for _, tc := range []struct {
		override any
		want     float64
	}{{nil, 7000000}, {0, 0}, {5000000, 5000000}} {
		testdb.Exec(t, tx, `UPDATE schedules SET minimal_dp=? WHERE id=?`, tc.override, id)
		rows, err := tx.Query(selectFull+` WHERE s.id=?`, id)
		if err != nil {
			t.Fatal(err)
		}
		if !rows.Next() {
			rows.Close()
			t.Fatal("schedule missing")
		}
		schedule, err := scanRow(rows)
		rows.Close()
		if err != nil {
			t.Fatal(err)
		}
		if got := schedule.ToPublic().EffectiveMinimalDP; got != tc.want {
			t.Fatalf("override %v: got %v want %v", tc.override, got, tc.want)
		}
	}
}

// Uses the project's opt-in rollback fixture; no schedule edits are committed.
func TestPrepareUpdateKeepsLatestSeatsAndBookingBrand(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	booking := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 20_000_000, Blocked: true})
	var brand int64
	if err := tx.QueryRow(`SELECT brand_id FROM schedules WHERE id=?`, booking.ScheduleID).Scan(&brand); err != nil {
		t.Fatal(err)
	}
	testdb.Exec(t, tx, `UPDATE schedules SET seat_total=45, seat_sisa=44 WHERE id=?`, booking.ScheduleID)
	oldTotal := 45
	inp := ScheduleInput{SeatTotal: 45, SeatSisa: 45, ExpectedSeatTotal: &oldTotal}
	if err := prepareScheduleUpdate(ctx, tx, booking.ScheduleID, &inp, &brand, brand); err != nil {
		t.Fatal(err)
	}
	if inp.SeatSisa != 44 {
		t.Fatalf("stale form restored seats: %d", inp.SeatSisa)
	}
	if err := prepareScheduleUpdate(ctx, tx, booking.ScheduleID, &inp, nil, brand+1); !errors.Is(err, ErrConflict) {
		t.Fatalf("brand with bookings: %v", err)
	}
	otherBrand := brand + 1
	if err := prepareScheduleUpdate(ctx, tx, booking.ScheduleID, &inp, &otherBrand, brand); !errors.Is(err, ErrNotFound) {
		t.Fatalf("tenant guard: %v", err)
	}
	oldTotal = 40
	if err := prepareScheduleUpdate(ctx, tx, booking.ScheduleID, &inp, nil, brand); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale capacity: %v", err)
	}
}
