package models

// TransactionWithParties adds party information to TransactionCore
type TransactionWithParties struct {
	TransactionCore
	Payer     Party                `json:"payer" yaml:"payer"`
	Payee     Party                `json:"payee" yaml:"payee"`
	Direction TransactionDirection `json:"direction" yaml:"direction"`
}

// NewTransactionWithParties creates a new TransactionWithParties instance
func NewTransactionWithParties() TransactionWithParties {
	return TransactionWithParties{
		TransactionCore: NewTransactionCore(),
		Direction:       DirectionUnknown,
	}
}

// NewTransactionWithPartiesFromCore creates a new TransactionWithParties from a TransactionCore
func NewTransactionWithPartiesFromCore(core TransactionCore) TransactionWithParties {
	return TransactionWithParties{
		TransactionCore: core,
		Direction:       DirectionUnknown,
	}
}

// IsDebit returns true if the transaction is a debit (outgoing money)
func (twp TransactionWithParties) IsDebit() bool {
	return twp.Direction == DirectionDebit || twp.Amount.IsNegative()
}

// IsCredit returns true if the transaction is a credit (incoming money)
func (twp TransactionWithParties) IsCredit() bool {
	return twp.Direction == DirectionCredit || (!twp.Amount.IsNegative() && twp.Direction != DirectionDebit)
}

// GetCounterparty returns the relevant party based on the transaction direction
// For debits, returns the payee (who receives the money)
// For credits, returns the payer (who sent the money)
func (twp TransactionWithParties) GetCounterparty() Party {
	if twp.IsDebit() {
		return twp.Payee
	}
	return twp.Payer
}

// GetCounterpartyName returns the name of the counterparty
func (twp TransactionWithParties) GetCounterpartyName() string {
	return twp.GetCounterparty().Name
}

// GetCounterpartyIBAN returns the IBAN of the counterparty
func (twp TransactionWithParties) GetCounterpartyIBAN() string {
	return twp.GetCounterparty().IBAN
}

// HasPayer returns true if the transaction has payer information
func (twp TransactionWithParties) HasPayer() bool {
	return !twp.Payer.IsEmpty()
}

// HasPayee returns true if the transaction has payee information
func (twp TransactionWithParties) HasPayee() bool {
	return !twp.Payee.IsEmpty()
}

// WithPayer sets the payer and returns a new TransactionWithParties
func (twp TransactionWithParties) WithPayer(payer Party) TransactionWithParties {
	twp.Payer = payer
	return twp
}

// WithPayee sets the payee and returns a new TransactionWithParties
func (twp TransactionWithParties) WithPayee(payee Party) TransactionWithParties {
	twp.Payee = payee
	return twp
}

// WithDirection sets the direction and returns a new TransactionWithParties
func (twp TransactionWithParties) WithDirection(direction TransactionDirection) TransactionWithParties {
	twp.Direction = direction
	return twp
}

// AsDebit sets the direction to debit and returns a new TransactionWithParties
func (twp TransactionWithParties) AsDebit() TransactionWithParties {
	twp.Direction = DirectionDebit
	return twp
}

// AsCredit sets the direction to credit and returns a new TransactionWithParties
func (twp TransactionWithParties) AsCredit() TransactionWithParties {
	twp.Direction = DirectionCredit
	return twp
}

// DeriveDirectionFromAmount sets the direction based on the amount sign
// Negative amounts are considered debits, positive amounts are credits
func (twp TransactionWithParties) DeriveDirectionFromAmount() TransactionWithParties {
	if twp.Amount.IsNegative() {
		twp.Direction = DirectionDebit
	} else if twp.Amount.IsPositive() {
		twp.Direction = DirectionCredit
	} else {
		twp.Direction = DirectionUnknown
	}
	return twp
}

// Equal returns true if two TransactionWithParties instances are equal
func (twp TransactionWithParties) Equal(other TransactionWithParties) bool {
	return twp.TransactionCore.Equal(other.TransactionCore) &&
		twp.Payer.Equal(other.Payer) &&
		twp.Payee.Equal(other.Payee) &&
		twp.Direction == other.Direction
}