package models

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Money represents a monetary value with currency
type Money struct {
	Amount   decimal.Decimal `json:"amount" yaml:"amount"`
	Currency string          `json:"currency" yaml:"currency"`
}

// NewMoney creates a new Money instance with the given amount and currency
func NewMoney(amount decimal.Decimal, currency string) Money {
	return Money{
		Amount:   amount,
		Currency: currency,
	}
}

// NewMoneyFromFloat creates a new Money instance from a float64 amount
// Note: Use this sparingly as float64 can introduce precision errors
func NewMoneyFromFloat(amount float64, currency string) Money {
	return Money{
		Amount:   decimal.NewFromFloat(amount),
		Currency: currency,
	}
}

// NewMoneyFromString creates a new Money instance from a string amount
func NewMoneyFromString(amount, currency string) (Money, error) {
	dec, err := decimal.NewFromString(amount)
	if err != nil {
		return Money{}, fmt.Errorf("invalid amount string '%s': %w", amount, err)
	}
	return Money{
		Amount:   dec,
		Currency: currency,
	}, nil
}

// Zero returns a Money instance with zero amount in the given currency
func ZeroMoney(currency string) Money {
	return Money{
		Amount:   decimal.Zero,
		Currency: currency,
	}
}

// IsZero returns true if the amount is zero
func (m Money) IsZero() bool {
	return m.Amount.IsZero()
}

// IsPositive returns true if the amount is positive
func (m Money) IsPositive() bool {
	return m.Amount.IsPositive()
}

// IsNegative returns true if the amount is negative
func (m Money) IsNegative() bool {
	return m.Amount.IsNegative()
}

// Abs returns the absolute value of the money amount
func (m Money) Abs() Money {
	return Money{
		Amount:   m.Amount.Abs(),
		Currency: m.Currency,
	}
}

// Neg returns the negated money amount
func (m Money) Neg() Money {
	return Money{
		Amount:   m.Amount.Neg(),
		Currency: m.Currency,
	}
}

// Add adds another Money value to this one
// Returns an error if currencies don't match
func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("cannot add different currencies: %s and %s", m.Currency, other.Currency)
	}
	return Money{
		Amount:   m.Amount.Add(other.Amount),
		Currency: m.Currency,
	}, nil
}

// Sub subtracts another Money value from this one
// Returns an error if currencies don't match
func (m Money) Sub(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("cannot subtract different currencies: %s and %s", m.Currency, other.Currency)
	}
	return Money{
		Amount:   m.Amount.Sub(other.Amount),
		Currency: m.Currency,
	}, nil
}

// Mul multiplies the money amount by a decimal factor
func (m Money) Mul(factor decimal.Decimal) Money {
	return Money{
		Amount:   m.Amount.Mul(factor),
		Currency: m.Currency,
	}
}

// Div divides the money amount by a decimal divisor
func (m Money) Div(divisor decimal.Decimal) Money {
	return Money{
		Amount:   m.Amount.Div(divisor),
		Currency: m.Currency,
	}
}

// String returns a string representation of the money value
func (m Money) String() string {
	return fmt.Sprintf("%s %s", m.Amount.StringFixed(2), m.Currency)
}

// StringFixed returns a string representation with fixed decimal places
func (m Money) StringFixed(places int32) string {
	return fmt.Sprintf("%s %s", m.Amount.StringFixed(places), m.Currency)
}

// Float64 returns the amount as a float64
// Note: This can introduce precision errors and should be used carefully
func (m Money) Float64() float64 {
	f, _ := m.Amount.Float64()
	return f
}

// Equal returns true if two Money values are equal (same amount and currency)
func (m Money) Equal(other Money) bool {
	return m.Amount.Equal(other.Amount) && m.Currency == other.Currency
}

// Compare compares two Money values
// Returns -1 if m < other, 0 if m == other, 1 if m > other
// Returns an error if currencies don't match
func (m Money) Compare(other Money) (int, error) {
	if m.Currency != other.Currency {
		return 0, fmt.Errorf("cannot compare different currencies: %s and %s", m.Currency, other.Currency)
	}
	return m.Amount.Cmp(other.Amount), nil
}
