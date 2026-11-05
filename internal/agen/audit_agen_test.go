package agen

import "testing"

// Audit probes: PASS confirms the behavior described, not correctness.
func TestAuditAgenBankAccountWithoutDigitsAccepted(t *testing.T) {
	req := AjukanPencairanRequest{Nominal: 500000, Rekening: Rekening{Bank: "BSI", Nomor: "---", AtasNama: "Audit"}}
	if err := req.normalize(); err != nil {
		t.Fatalf("probe changed: %v", err)
	}
	t.Log("CONFIRMED: bank account --- passes payout validation")
}
