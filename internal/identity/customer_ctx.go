package identity

import "context"

type customerCtxKey int

const (
	customerAccountKey customerCtxKey = iota
	customerBrandKey
)

// WithCustomer menyimpan akun pelanggan (dan brand-nya) di context. Dipanggil middleware
// sesi pelanggan; paket lain (mis. media) membacanya tanpa mengimpor paket customer.
func WithCustomer(ctx context.Context, accountID, brandID int64) context.Context {
	ctx = context.WithValue(ctx, customerAccountKey, accountID)
	return context.WithValue(ctx, customerBrandKey, brandID)
}

func GetCustomerAccountID(ctx context.Context) int64 {
	v, _ := ctx.Value(customerAccountKey).(int64)
	return v
}

func GetCustomerBrandID(ctx context.Context) int64 {
	v, _ := ctx.Value(customerBrandKey).(int64)
	return v
}
