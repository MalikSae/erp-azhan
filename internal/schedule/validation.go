package schedule

import (
	"errors"
	"fmt"
	"math"
)

var ErrConflict = errors.New("perubahan paket ditolak")

func remainingAfterCapacityChange(total, remaining, nextTotal int) (int, error) {
	allocated := total - remaining
	if remaining < 0 || remaining > total || nextTotal <= 0 || nextTotal < allocated {
		return 0, fmt.Errorf("%w: kapasitas tidak boleh kurang dari kursi teralokasi", ErrConflict)
	}
	return nextTotal - allocated, nil
}

func validateDP(dp *float64, prices ...float64) error {
	if dp == nil {
		return nil
	}
	if math.IsNaN(*dp) || math.IsInf(*dp, 0) || *dp < 0 {
		return errors.New("minimal_dp harus berupa angka tidak negatif")
	}
	for _, price := range prices {
		if *dp > price {
			return errors.New("minimal_dp tidak boleh melebihi harga kamar termurah")
		}
	}
	return nil
}
