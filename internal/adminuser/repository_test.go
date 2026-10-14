package adminuser

import "testing"

func strPtr(s string) *string { return &s }

func TestSameSet(t *testing.T) {
	cases := []struct {
		a, b []string
		want bool
	}{
		{[]string{"cmo", "super_admin_grup"}, []string{"super_admin_grup", "cmo"}, true},
		{[]string{"cmo"}, []string{"cmo", "finance"}, false},
		{[]string{}, []string{}, true},
		{[]string{"cmo"}, []string{"finance"}, false},
	}
	for _, c := range cases {
		if got := sameSet(c.a, c.b); got != c.want {
			t.Fatalf("sameSet(%v,%v)=%v want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestNormalizeOptional(t *testing.T) {
	if normalizeOptional(nil) != nil {
		t.Fatal("nil harus tetap nil")
	}
	if normalizeOptional(strPtr("   ")) != nil {
		t.Fatal("string kosong harus jadi nil")
	}
	if got := normalizeOptional(strPtr("  Direktur ")); got == nil || *got != "Direktur" {
		t.Fatalf("trim gagal: %v", got)
	}
}

func TestValidateUpsert(t *testing.T) {
	valid := func() UpsertInput {
		return UpsertInput{DisplayName: "Haikal", Email: " Haikal@Gmail.com ", Roles: []string{"direktur", "direktur", " "}}
	}

	in := valid()
	if msg := validateUpsert(&in); msg != "" {
		t.Fatalf("input valid ditolak: %s", msg)
	}
	if in.Email != "haikal@gmail.com" {
		t.Fatalf("email harus di-trim & lowercase, dapat %q", in.Email)
	}
	if len(in.Roles) != 1 {
		t.Fatalf("role harus dinormalisasi (dedupe, buang kosong), dapat %v", in.Roles)
	}
	if in.Keanggotaan == nil {
		t.Fatal("keanggotaan nil harus jadi slice kosong")
	}

	bad := []struct {
		name   string
		mutate func(*UpsertInput)
	}{
		{"nama terlalu pendek", func(u *UpsertInput) { u.DisplayName = "H" }},
		{"email salah", func(u *UpsertInput) { u.Email = "bukan-email" }},
		{"tanpa role", func(u *UpsertInput) { u.Roles = []string{"  "} }},
		{"no hp terlalu panjang", func(u *UpsertInput) { u.NoHP = strPtr("0812345678901234567890123456789012") }},
	}
	for _, b := range bad {
		in := valid()
		b.mutate(&in)
		if msg := validateUpsert(&in); msg == "" {
			t.Fatalf("%s: seharusnya ditolak", b.name)
		}
	}
}
