package adminuser

import (
	"testing"

	"erp-azhan/api/internal/akses"
)

func strPtr(s string) *string { return &s }

func TestNormalizeOptional(t *testing.T) {
	if normalizeOptional(nil) != nil {
		t.Fatal("nil harus tetap nil")
	}
	if normalizeOptional(strPtr("   ")) != nil {
		t.Fatal("string kosong harus jadi nil")
	}
	if got := normalizeOptional(strPtr("  0812 ")); got == nil || *got != "0812" {
		t.Fatalf("trim gagal: %v", got)
	}
}

func TestValidateUpsert(t *testing.T) {
	valid := func() UpsertInput {
		return UpsertInput{DisplayName: "Haikal", Email: " Haikal@Gmail.com ", Akses: akses.Tim,
			Permissions: []string{"booking.view", "booking.view", " "}}
	}

	in := valid()
	if msg := validateUpsert(&in); msg != "" {
		t.Fatalf("input valid ditolak: %s", msg)
	}
	if in.Email != "haikal@gmail.com" {
		t.Fatalf("email harus di-trim & lowercase, dapat %q", in.Email)
	}
	if len(in.Permissions) != 1 {
		t.Fatalf("izin harus dinormalisasi (dedupe, buang kosong), dapat %v", in.Permissions)
	}

	super := UpsertInput{DisplayName: "Haikal", Email: "haikal@gmail.com", Akses: akses.SuperAdmin}
	if msg := validateUpsert(&super); msg != "" {
		t.Fatalf("Super Admin tanpa izin harus sah: %s", msg)
	}

	bad := []struct {
		name   string
		mutate func(*UpsertInput)
	}{
		{"nama terlalu pendek", func(u *UpsertInput) { u.DisplayName = "H" }},
		{"email salah", func(u *UpsertInput) { u.Email = "bukan-email" }},
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
