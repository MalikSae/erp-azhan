package testdb

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

// CloneSchedule menyalin satu jadwal (keberangkatan lain dengan data sama) dan mengembalikan id barunya.
func CloneSchedule(t *testing.T, tx *sql.Tx, scheduleID int64) int64 {
	t.Helper()
	rows, err := tx.QueryContext(context.Background(), `
		SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'schedules' AND COLUMN_NAME <> 'id' AND EXTRA NOT LIKE '%GENERATED%'
		ORDER BY ORDINAL_POSITION`)
	if err != nil {
		t.Fatalf("kolom schedules: %v", err)
	}
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatal(err)
		}
		cols = append(cols, "`"+c+"`")
	}
	rows.Close()
	list := strings.Join(cols, ",")
	res := mustExec(t, tx, "INSERT INTO schedules ("+list+") SELECT "+list+" FROM schedules WHERE id = ?", scheduleID)
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}
