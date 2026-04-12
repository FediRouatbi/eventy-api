package events

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/stripe/stripe-go/v81"
)

type UnsupportedCurrencyError struct {
	Currency string
}

func (e UnsupportedCurrencyError) Error() string {
	currency := strings.ToUpper(strings.TrimSpace(e.Currency))
	if currency == "" {
		return ErrUnsupportedCurrency.Error()
	}

	return fmt.Sprintf("currency %s is not supported by stripe for this account", currency)
}

func (e UnsupportedCurrencyError) Unwrap() error {
	return ErrUnsupportedCurrency
}

var invalidCurrencyRegex = regexp.MustCompile(`(?i)\binvalid currency:\s*([a-z]{3})\b`)

func asUnsupportedCurrencyError(err error) (UnsupportedCurrencyError, bool) {
	var stripeErr *stripe.Error
	if !errors.As(err, &stripeErr) {
		return UnsupportedCurrencyError{}, false
	}

	// Stripe uses invalid_request_error with a param like:
	// "line_items[0][price_data][currency]"
	if !strings.Contains(strings.ToLower(stripeErr.Param), "currency") {
		return UnsupportedCurrencyError{}, false
	}

	matches := invalidCurrencyRegex.FindStringSubmatch(stripeErr.Msg)
	if len(matches) < 2 {
		return UnsupportedCurrencyError{}, false
	}

	return UnsupportedCurrencyError{Currency: matches[1]}, true
}
