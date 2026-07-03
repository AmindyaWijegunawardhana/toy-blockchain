package ledger

// Transaction represents a simple value transfer.
type Transaction struct {
	Sender    string  `json:"sender"`
	Recipient string  `json:"recipient"`
	Amount    float64 `json:"amount"`
}

// NewTransaction creates a new transaction instance.
func NewTransaction(sender, recipient string, amount float64) Transaction {
	return Transaction{
		Sender:    sender,
		Recipient: recipient,
		Amount:    amount,
	}
}
