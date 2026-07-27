package ledger

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"strconv"
)

// Transaction represents a signed value transfer within the ledger.
type Transaction struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Amount    int64  `json:"amount"`
	Signature string `json:"signature,omitempty"`
}

// NewTransaction constructs a new transaction instance.
func NewTransaction(sender, recipient string, amount int64) Transaction {
	return Transaction{
		Sender:    sender,
		Recipient: recipient,
		Amount:    amount,
	}
}

// GetHash generates a clean SHA-256 hash of the transaction components.
func (tx *Transaction) GetHash() []byte {
	record := strconv.FormatInt(tx.Amount, 10) + tx.Sender + tx.Recipient
	hash := sha256.Sum256([]byte(record))
	return hash[:]
}

// Sign uses the sender's private key to sign the transaction hash.
func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey) error {
	txHash := tx.GetHash()
	r, s, err := ecdsa.Sign(rand.Reader, privKey, txHash)
	if err != nil {
		return err
	}

	// Secure padding: Allocate exactly 32 bytes for both R and S coordinates
	rBuf := make([]byte, 32)
	sBuf := make([]byte, 32)
	r.FillBytes(rBuf)
	s.FillBytes(sBuf)

	signatureBytes := append(rBuf, sBuf...)
	tx.Signature = hex.EncodeToString(signatureBytes)
	return nil
}

// Verify checks if the transaction signature matches the sender's public key hex string.
func (tx *Transaction) Verify() bool {
	if tx.Sender == "faucet" || tx.Sender == "system" {
		return true
	}
	if tx.Signature == "" {
		return false
	}

	pubKeyBytes, err := hex.DecodeString(tx.Sender)
	if err != nil || len(pubKeyBytes) != 64 { // Must be exactly 32 + 32 bytes
		return false
	}

	// Safely split fixed 32-byte chunks
	xBytes, yBytes := pubKeyBytes[:32], pubKeyBytes[32:]
	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)

	curve := elliptic.P256()
	pubKey := &ecdsa.PublicKey{Curve: curve, X: x, Y: y}

	sigBytes, err := hex.DecodeString(tx.Signature)
	if err != nil || len(sigBytes) != 64 { // Must be exactly 32 + 32 bytes
		return false
	}

	rBytes, sBytes := sigBytes[:32], sigBytes[32:]
	r := new(big.Int).SetBytes(rBytes)
	s := new(big.Int).SetBytes(sBytes)

	return ecdsa.Verify(pubKey, tx.GetHash(), r, s)
}

// GenerateKeyPair outputs a valid ECDSA wallet key pair with fixed 32-byte field padding.
func GenerateKeyPair() (*ecdsa.PrivateKey, string, error) {
	curve := elliptic.P256()
	privKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, "", err
	}

	// Secure padding: Allocate exactly 32 bytes for X and Y coordinates
	xBuf := make([]byte, 32)
	yBuf := make([]byte, 32)
	privKey.PublicKey.X.FillBytes(xBuf)
	privKey.PublicKey.Y.FillBytes(yBuf)

	pubKeyBytes := append(xBuf, yBuf...)
	pubKeyHex := hex.EncodeToString(pubKeyBytes)

	return privKey, pubKeyHex, nil
}
