package selfbooking

import "errors"

var ErrInfantUnavailable = errors.New("harga infant belum tersedia, silakan hubungi admin travel")
var ErrInvalidDP = errors.New("konfigurasi DP paket tidak valid, silakan hubungi admin travel")
