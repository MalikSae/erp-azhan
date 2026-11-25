package shared

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

// IsMySQLErr: err (atau galat yang dibungkusnya) adalah galat MySQL bernomor code.
func IsMySQLErr(err error, code uint16) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == code
}

// IsFKDipakai: baris tidak bisa dihapus karena masih dirujuk foreign key (MySQL 1451).
func IsFKDipakai(err error) bool { return IsMySQLErr(err, 1451) }
