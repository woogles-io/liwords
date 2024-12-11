package entitlements

import (
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestPaymentsUpToDate(t *testing.T) {
	is := is.New(t)
	// 2024-11-25T03:04:05Z
	now := time.Date(2024, 11, 25, 3, 4, 5, 0, time.UTC)

	testcases := []struct {
		lastPaymentDate string
		paymentUpToDate bool
		expectedAllowed bool
	}{
		{"2024-11-20T00:00:00Z", true, true},
		{"2024-11-20T00:00:00Z", false, false}, // it's an attempt date, not payment date, so we say false.
		{"2024-10-20T00:00:00Z", true, false},  /// too long ago, today is 11-25
	}
	for _, tc := range testcases {

		dt, err := time.Parse(time.RFC3339, tc.lastPaymentDate)
		is.NoErr(err)
		utd, _ := paymentsUpToDate(dt, tc.paymentUpToDate, now)
		is.Equal(utd, tc.expectedAllowed)
	}

}
