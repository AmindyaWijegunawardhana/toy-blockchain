package ledger

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
)

// Transaction represents a transfer of value.
type Transaction struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Amount    int64  `json:"amount"`
	Signature string `json:"signature,omitempty"`
	PubKey    string `json:"pub_key,omitempty"`
}

// Wallet holds ECDSA key pairs for signing transactions.
type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  []byte
}

// NewWallet creates a new cryptographic wallet.
func NewWallet() (*Wallet, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	pub := elliptic.Marshal(elliptic.P256(), priv.PublicKey.X, priv.PublicKey.Y)
	return &Wallet{PrivateKey: priv, PublicKey: pub}, nil
}

// Address returns the hex-encoded string of the public key.
func (w *Wallet) Address() string {
	return hex.EncodeToString(w.PublicKey)
}

// NewTransaction creates an unsigned transaction (useful for system/coinbase/genesis transactions).
func NewTransaction(sender, recipient string, amount int64) Transaction {
	return Transaction{
		Sender:    sender,
		Recipient: recipient,
		Amount:    amount,
	}
}

// GetHash computes the SHA-256 hash as raw []byte for Merkle trees and cryptographic operations.
func (tx Transaction) GetHash() []byte {
	record := fmt.Sprintf("%s:%s:%d:%s:%s", tx.Sender, tx.Recipient, tx.Amount, tx.Signature, tx.PubKey)
	hash := sha256.Sum256([]byte(record))
	return hash[:]
}

// Hash returns the hex-encoded SHA-256 hash string.
func (tx Transaction) Hash() string {
	return hex.EncodeToString(tx.GetHash())
}

// SignTransaction creates a signed transaction.
func SignTransaction(w *Wallet, recipient string, amount int64) (*Transaction, error) {
	if w == nil || w.PrivateKey == nil {
		return nil, errors.New("invalid wallet")
	}

	tx := &Transaction{
		Sender:    w.Address(),
		Recipient: recipient,
		Amount:    amount,
		PubKey:    w.Address(),
	}

	msg := fmt.Sprintf("%s:%s:%d", tx.Sender, tx.Recipient, tx.Amount)
	hash := sha256.Sum256([]byte(msg))

	r, s, err := ecdsa.Sign(rand.Reader, w.PrivateKey, hash[:])
	if err != nil {
		return nil, err
	}

	rBytes := r.Bytes()
	sBytes := s.Bytes()
	sigBytes := append(rBytes, sBytes...)
	tx.Signature = hex.EncodeToString(sigBytes)

	return tx, nil
}

// VerifyTransaction verifies the ECDSA signature of a transaction.
func VerifyTransaction(tx *Transaction) error {
	if tx.Sender == "" {
		// System / Genesis transaction
		return nil
	}

	if tx.Signature == "" || tx.PubKey == "" {
		return errors.New("missing signature or public key")
	}

	pubBytes, err := hex.DecodeString(tx.PubKey)
	if err != nil {
		return errors.New("invalid public key encoding")
	}

	x, y := elliptic.Unmarshal(elliptic.P256(), pubBytes)
	if x == nil || y == nil {
		return errors.New("failed to unmarshal public key")
	}
	pubKey := ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}

	sigBytes, err := hex.DecodeString(tx.Signature)
	if err != nil || len(sigBytes) < 2 {
		return errors.New("invalid signature encoding")
	}

	half := len(sigBytes) / 2
	r := new(big.Int).SetBytes(sigBytes[:half])
	s := new(big.Int).SetBytes(sigBytes[half:])

	msg := fmt.Sprintf("%s:%s:%d", tx.Sender, tx.Recipient, tx.Amount)
	hash := sha256.Sum256([]byte(msg))

	if !ecdsa.Verify(&pubKey, hash[:], r, s) {
		return errors.New("signature verification failed")
	}

	return nil
}
