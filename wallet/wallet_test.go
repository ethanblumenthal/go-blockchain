package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
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

func TestSignCryptoParams(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	spew.Dump(privKey)

	msg := []byte("This is a test message")

	sig, err := Sign(msg, privKey)
	if err != nil {
		t.Fatal(err)
	}

	if len(sig) != crypto.SignatureLength {
		t.Fatal(fmt.Errorf("wrong size for signature: got %d, want %d", len(sig), crypto.SignatureLength))
	}

	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:64])
	v := new(big.Int).SetBytes([]byte{sig[64]})

	spew.Dump(r, s, v)
}

func TestSign(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	pubKey := privKey.PublicKey
	pubKeyBytes := elliptic.Marshal(crypto.S256(), pubKey.X, pubKey.Y)
	pubKeyBytesHash := crypto.Keccak256(pubKeyBytes[1:])

	account := common.BytesToAddress(pubKeyBytesHash[12:])

	msg := []byte("This is a test message")

	sig, err := Sign(msg, privKey)
	if err != nil {
		t.Fatal(err)
	}

	recoveredPubKey, err := Verify(msg, sig)
	if err != nil {
		t.Fatal(err)
	}

	recoveredPubKeyBytes := elliptic.Marshal(crypto.S256(), recoveredPubKey.X, recoveredPubKey.Y)
	recoveredPubKeyBytesHash := crypto.Keccak256(recoveredPubKeyBytes[1:])
	recoveredAccount := common.BytesToAddress(recoveredPubKeyBytesHash[12:])

	if account.Hex() != recoveredAccount.Hex() {
		t.Fatalf("msg was signed by account %s but signature recovery produced an account %s", account.Hex(), recoveredAccount.Hex())
	}
}

func TestSignTxWithKeystoreAccount(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "wallet_test")
	if err != nil {
		t.Fatal(err)
	}
	defer fs.RemoveDir(tmpDir)

	account1, err := NewKeystoreAccount(tmpDir, testKeystoreAccountsPwd)
	if err != nil {
		t.Error(err)
		return
	}

	account2, err := NewKeystoreAccount(tmpDir, testKeystoreAccountsPwd)
	if err != nil {
		t.Error(err)
		return
	}

	tx := database.NewTx(account1, account2, 100, 1, "")

	signedTx, err := SignTxWithKeystoreAccount(tx, account1, testKeystoreAccountsPwd, GetKeystoreDirPath(tmpDir))
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
		t.Error(err)
		return
	}

	account2, err := NewKeystoreAccount(tmpDir, testKeystoreAccountsPwd)
	if err != nil {
		t.Error(err)
		return
	}

	forgedTx := database.NewTx(account2, hacker, 100, 1, "")

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

	if ok {
		t.Fatal("the TX 'from' attribute was forged and should have not be authentic")
	}
}