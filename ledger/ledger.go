package ledger

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Transaction represents an asset transfer between accounts.
type Transaction struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Amount    int64  `json:"amount"`
	Signature string `json:"signature"`
	PubBytes  string `json:"public_key"` // Hex-encoded Ed25519 sender public key
}

// NewTransaction creates an unsigned transaction.
func NewTransaction(sender, recipient string, amount int64) Transaction {
	return Transaction{
		Sender:    sender,
		Recipient: recipient,
		Amount:    amount,
	}
}

// GetHash calculates the SHA-256 hash of the transaction data fields.
func (tx *Transaction) GetHash() []byte {
	record := fmt.Sprintf("%s:%s:%d", tx.Sender, tx.Recipient, tx.Amount)
	hash := sha256.Sum256([]byte(record))
	return hash[:]
}

// GenerateKeyPair generates a fresh Ed25519 public/private keypair.
func GenerateKeyPair() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return priv, pub, nil
}

// Sign signs the transaction hash using an Ed25519 private key.
func (tx *Transaction) Sign(priv ed25519.PrivateKey) error {
	hash := tx.GetHash()
	sig := ed25519.Sign(priv, hash)
	tx.Signature = hex.EncodeToString(sig)

	pubKey := priv.Public().(ed25519.PublicKey)
	tx.PubBytes = hex.EncodeToString(pubKey)
	return nil
}

// Verify checks the cryptographic Ed25519 signature and enforces sender identity matches public key.
func (tx *Transaction) Verify() bool {
	if tx.Sender == "faucet" || tx.Sender == "system" {
		return true // Allow protocol minting / Genesis transactions
	}

	if tx.Signature == "" || tx.PubBytes == "" {
		return false
	}

	// 🔒 Security Gate: The public key attached MUST match the declared sender address!
	if tx.Sender != tx.PubBytes {
		return false
	}

	sigBytes, err := hex.DecodeString(tx.Signature)
	if err != nil {
		return false
	}

	pubBytes, err := hex.DecodeString(tx.PubBytes)
	if err != nil {
		return false
	}

	if len(pubBytes) != ed25519.PublicKeySize || len(sigBytes) != ed25519.SignatureSize {
		return false
	}

	pubKey := ed25519.PublicKey(pubBytes)
	hash := tx.GetHash()

	return ed25519.Verify(pubKey, hash, sigBytes)
}
