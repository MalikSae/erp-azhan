package rbac

import "time"

// Role merepresentasikan record di tabel roles beserta daftar permission-nya.
type Role struct {
	ID          int64     `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	IsSystem    bool      `json:"is_system"`
	Scope       string    `json:"scope"` // 'holding' | 'brand' | 'semua'
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
}

// Permission merepresentasikan record di tabel permissions.
type Permission struct {
	ID          int64  `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
}

// UserAccess hasil resolusi role & permission milik satu admin user.
type UserAccess struct {
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	PermVersion int      `json:"perm_version"`
}

// SetUserRolesRequest payload penggantian role user.
type SetUserRolesRequest struct {
	Roles []string `json:"roles"`
}
