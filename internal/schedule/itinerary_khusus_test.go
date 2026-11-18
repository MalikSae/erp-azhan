package schedule

import (
	"encoding/json"
	"testing"
)

func TestItineraryKhususDecode(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantNil bool
		days    int
	}{
		{"absent", `{"jadwal_nama":"x"}`, true, 0},
		{"null", `{"jadwal_nama":"x","itinerary_khusus":null}`, true, 0},
		{"object", `{"itinerary_khusus":{"title":"","days":[{"title":"H1","location":"Jeddah","activities":[{"time":"","text":"Tiba"}]}]}}`, false, 1},
		{"object kosong", `{"itinerary_khusus":{}}`, false, 0},
	}
	for _, c := range cases {
		var req CreateScheduleRequest
		if err := json.Unmarshal([]byte(c.body), &req); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if (req.ItineraryKhusus == nil) != c.wantNil {
			t.Fatalf("%s: nil=%v want %v", c.name, req.ItineraryKhusus == nil, c.wantNil)
		}
		if req.ItineraryKhusus != nil && len(req.ItineraryKhusus.Days) != c.days {
			t.Fatalf("%s: days=%d want %d", c.name, len(req.ItineraryKhusus.Days), c.days)
		}
	}
}

func TestItineraryKhususFlagJSON(t *testing.T) {
	b, _ := json.Marshal(Schedule{ItineraryKhusus: true})
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["itinerary_khusus"] != true {
		t.Fatalf("itinerary_khusus = %v", m["itinerary_khusus"])
	}
}
