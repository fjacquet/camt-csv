package models

import (
	"time"

	"github.com/google/uuid"
)

// TransactionCore contains essential transaction data
type TransactionCore struct {
	ID          string    `json:"id" yaml:"id"`
	Date        time.Time `json:"date" yaml:"date"`
	ValueDate   time.Time `json:"value_date" yaml:"value_date"`
	Amount      Money     `json:"amount" yaml:"amount"`
	Description string    `json:"description" yaml:"description"`
	Status      string    `json:"status" yaml:"status"`
	Reference   string    `json:"reference" yaml:"reference"`
}

// NewTransactionCore creates a new TransactionCore instance with a generated ID
func NewTransactionCore() TransactionCore {
	return TransactionCore{
		ID:     uuid.New().String(),
		Status: StatusCompleted, // Default status
	}
}

// NewTransactionCoreWithID creates a new TransactionCore instance with a specific ID
func NewTransactionCoreWithID(id string) TransactionCore {
	return TransactionCore{
		ID:     id,
		Status: StatusCompleted, // Default status
	}
}

// IsEmpty returns true if the transaction core has minimal data
func (tc TransactionCore) IsEmpty() bool {
	return tc.Date.IsZero() && tc.Amount.IsZero() && tc.Description == ""
}

// HasValidDate returns true if the transaction has a valid date
func (tc TransactionCore) HasValidDate() bool {
	return !tc.Date.IsZero()
}

// HasValidValueDate returns true if the transaction has a valid value date
func (tc TransactionCore) HasValidValueDate() bool {
	return !tc.ValueDate.IsZero()
}

// GetEffectiveDate returns the value date if available, otherwise the transaction date
func (tc TransactionCore) GetEffectiveDate() time.Time {
	if tc.HasValidValueDate() {
		return tc.ValueDate
	}
	return tc.Date
}

// IsCompleted returns true if the transaction status is completed
func (tc TransactionCore) IsCompleted() bool {
	return tc.Status == StatusCompleted
}

// IsPending returns true if the transaction status is pending
func (tc TransactionCore) IsPending() bool {
	return tc.Status == StatusPending
}

// IsFailed returns true if the transaction status is failed
func (tc TransactionCore) IsFailed() bool {
	return tc.Status == StatusFailed
}

// WithDate sets the transaction date and returns a new TransactionCore
func (tc TransactionCore) WithDate(date time.Time) TransactionCore {
	tc.Date = date
	return tc
}

// WithValueDate sets the value date and returns a new TransactionCore
func (tc TransactionCore) WithValueDate(valueDate time.Time) TransactionCore {
	tc.ValueDate = valueDate
	return tc
}

// WithAmount sets the amount and returns a new TransactionCore
func (tc TransactionCore) WithAmount(amount Money) TransactionCore {
	tc.Amount = amount
	return tc
}

// WithDescription sets the description and returns a new TransactionCore
func (tc TransactionCore) WithDescription(description string) TransactionCore {
	tc.Description = description
	return tc
}

// WithStatus sets the status and returns a new TransactionCore
func (tc TransactionCore) WithStatus(status string) TransactionCore {
	tc.Status = status
	return tc
}

// WithReference sets the reference and returns a new TransactionCore
func (tc TransactionCore) WithReference(reference string) TransactionCore {
	tc.Reference = reference
	return tc
}

// Equal returns true if two TransactionCore instances are equal
func (tc TransactionCore) Equal(other TransactionCore) bool {
	return tc.ID == other.ID &&
		tc.Date.Equal(other.Date) &&
		tc.ValueDate.Equal(other.ValueDate) &&
		tc.Amount.Equal(other.Amount) &&
		tc.Description == other.Description &&
		tc.Status == other.Status &&
		tc.Reference == other.Reference
}
