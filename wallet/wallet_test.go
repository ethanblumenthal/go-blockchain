package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethanblumenthal/golang-blockchain/database"
	"github.com/ethanblumenthal/golang-blockchain/fs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const testKeystoreAccountsPwd = "security123"

func TestSignCyrptoParams(t *testing.T) {
	// Generate key on the fly
	privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	spew.Dump(privKey)

	// Prepare a message to digitally sign
	msg := []byte("This is a test message.")

	// Sign it
	sig, err := Sign(msg, privKey)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the length is 65 bytes
	if len(sig) != crypto.SignatureLength {
		t.Fatalf("wrong size for signature: got %d, want %d", len(sig), crypto.SignatureLength)
	}

	// Print the 3 required Ethereum signature crypto values
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:64])
	v := new(big.Int).SetBytes([]byte{sig[64]})

	spew.Dump(r, s, v)
}

func TestSign(t *testing.T) {
	// Generate private key on the fly
	privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// Convert the public key to bytes with elliptic curve settings
	pubKey := privKey.PublicKey
	pubKeyBytes := elliptic.Marshal(crypto.S256(), pubKey.X, pubKey.Y)

	// Hash the public key to 32 bytes
	pubKeyBytesHash := crypto.Keccak256(pubKeyBytes[1:])

	// The last 20 bytes of the public key hash will be the username
	account := common.BytesToAddress(pubKeyBytesHash[12:])

	msg := []byte("This is a test message")

	// Sign the message (generate message signature)
	sig, err := Sign(msg, privKey)
	if err != nil {
		t.Fatal(err)
	}

	// Recover the public key from the signature
	recoveredPubKey, err := Verify(msg, sig)
	if err != nil {
		t.Fatal(err)
	}

	// Convert the public key to a username again
	recoveredPubKeyBytes := elliptic.Marshal(crypto.S256(), recoveredPubKey.X, recoveredPubKey.Y)
	recoveredPubKeyBytesHash := crypto.Keccak256(recoveredPubKeyBytes[1:])
	recoveredAccount := common.BytesToAddress(recoveredPubKeyBytesHash[12:])

	if account.Hex() != recoveredAccount.Hex() {
		t.Fatalf(
			"msg was signed by account %s but signature recovered produced account %s",
			account.Hex(),
			recoveredAccount.Hex(),
		)
	}
}

func TestSignTxWithKeystoreAccount(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "wallet_test")
	if err != nil {
		t.Fatal(err)
	}
	defer fs.RemoveDir(tmpDir)

	acc1, err := NewKeystoreAccount(tmpDir, testKeystoreAccountsPwd)
	if err != nil {
		t.Fatal(err)
		return
	}

	acc2, err := NewKeystoreAccount(tmpDir, testKeystoreAccountsPwd)
	if err != nil {
		t.Fatal(err)
		return
	}

	tx := database.NewTx(acc1, acc2, 100, "")

	signedTx, err := SignTxWithKeystoreAccount(tx, acc1, testKeystoreAccountsPwd, GetKeystoreDirPath(tmpDir))
	if err != nil {
		t.Error(err)
		return
	}

	spew.Dump(signedTx.Encode())

	ok, err := signedTx.IsAuthentic()
	if err != nil {
		t.Error(err)
		return
	}

	if !ok {
		t.Fatal("the TX was signed by 'from' account and should have been authentic")
	}
}

func TestSignForgedTxWithKeystoreAccount(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "wallet_test")
	if err != nil {
		t.Fatal(err)
	}
	defer fs.RemoveDir(tmpDir)

	hacker, err := NewKeystoreAccount(tmpDir, testKeystoreAccountsPwd)
	if err != nil {
		t.Fatal(err)
		return
	}

	account, err := NewKeystoreAccount(tmpDir, testKeystoreAccountsPwd)
	if err != nil {
		t.Fatal(err)
		return
	}

	forgedTx := database.NewTx(account, hacker, 100, "")

	signedTx, err := SignTxWithKeystoreAccount(forgedTx, hacker, testKeystoreAccountsPwd, GetKeystoreDirPath(tmpDir))
	if err != nil {
		t.Error(err)
		return
	}

	ok, err := signedTx.IsAuthentic()
	if err != nil {
		t.Error(err)
		return
	}

	if !ok {
		t.Fatal("the TX 'from' attribute was forged and should have not be authentic")
	}
}