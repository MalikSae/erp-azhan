package komisi

import (
	"context"
	"database/sql"
	"time"
)

type CashbackMutation struct {
	BookingID int64     `json:"booking_id"`
	Nominal   float64   `json:"nominal"`
	Jenis     string    `json:"jenis"`
	CreatedAt time.Time `json:"created_at"`
}
type CashbackView struct {
	Balance float64            `json:"balance"`
	Items   []CashbackMutation `json:"items"`
}

func ReadCashback(ctx context.Context, db *sql.DB, id int64) (*CashbackView, error) {
	saldo, err := HitungSaldo(ctx, db, id)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `SELECT booking_id,nominal,'diterima',created_at FROM transaksi_komisi WHERE jamaah_penerima_id=? AND jenis='cashback' UNION ALL SELECT booking_id,-nominal,'dipakai',created_at FROM pemakaian_cashback WHERE jamaah_id=? ORDER BY created_at DESC`, id, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	v := &CashbackView{Balance: saldo.KreditCashback, Items: []CashbackMutation{}}
	for rows.Next() {
		var m CashbackMutation
		if err = rows.Scan(&m.BookingID, &m.Nominal, &m.Jenis, &m.CreatedAt); err != nil {
			return nil, err
		}
		v.Items = append(v.Items, m)
	}
	return v, rows.Err()
}
