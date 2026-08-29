package crmdeal

type JamaahInput struct {
	ID          *int64  `json:"id,omitempty"`
	NamaLengkap string  `json:"nama_lengkap"`
	NoHP        string  `json:"no_hp"`
	Email       *string `json:"email,omitempty"`
	Alamat      *string `json:"alamat,omitempty"`
}

type DealRequest struct {
	BrandID           *int64      `json:"brand_id,omitempty"`
	CRMLeadID         string      `json:"crm_lead_id"`
	Jamaah            JamaahInput `json:"jamaah"`
	ScheduleID        int64       `json:"schedule_id"`
	RoomType          string      `json:"room_type"`
	Pax               int         `json:"pax"`
	CommitmentType    string      `json:"commitment_type"`
	SeatHoldExpiresAt *string     `json:"seat_hold_expires_at,omitempty"`
	PaymentAmount     *float64    `json:"payment_amount,omitempty"`
	PaymentMethod     *string     `json:"payment_method,omitempty"`
	PaymentDate       *string     `json:"payment_date,omitempty"`
	PaymentProofURL   *string     `json:"payment_proof_url,omitempty"`
}

type DealResponse struct {
	Status            string  `json:"status"`
	CommitmentType    string  `json:"commitment_type"`
	DealSubstatus     string  `json:"deal_substatus"`
	JamaahID          int64   `json:"jamaah_id"`
	BookingID         int64   `json:"booking_id"`
	BookingCode       string  `json:"booking_code"`
	BookingStatus     string  `json:"booking_status"`
	PaymentID         *int64  `json:"payment_id,omitempty"`
	SeatHoldExpiresAt *string `json:"seat_hold_expires_at,omitempty"`
}
