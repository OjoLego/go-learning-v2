package validator

import "testing"

func TestValidateStruct_TransactionInvalidAmount(t *testing.T) {
	req := struct {
		Amount float64 `validate:"required,gt=0"`
	}{Amount: -5}
	errs := ValidateStruct(req)
	if len(errs) == 0 {
		t.Fatal("expected validation errors")
	}
}
