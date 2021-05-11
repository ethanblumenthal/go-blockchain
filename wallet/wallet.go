package wallet

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/ethanblumenthal/golang-blockchain/database"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const keystoreDirName = "keystore"
const Account1 = "0x22ba1F80452E6220c7cc6ea2D1e3EEDDaC5F694A"
const Account2 = "0x21973d33e048f5ce006fd7b41f51725c30e4b76b"
const Account3 = "0x84470a31D271ea400f34e7A697F36bE0e866a716"

func GetKeystoreDirPath(dataDir string) string {
	return filepath.Join(dataDir, keystoreDirName)
}

func NewKeystoreAccount(dataDir string, password string) (common.Address, error) {
	ks := keystore.NewKeyStore(GetKeystoreDirPath(dataDir), keystore.StandardScryptN, keystore.StandardScryptP)
	account, err := ks.NewAccount(password)
	if err != nil {
		return common.Address{}, err
	}

	return account.Address, nil
}

func SignTxWithKeystoreAccount(tx database.Tx, account common.Address, pwd string, keystoreDir string) (database.SignedTx, error) {
	ks := keystore.NewKeyStore(keystoreDir, keystore.StandardScryptN, keystore.StandardScryptP)
	ksAccount, err := ks.Find(accounts.Account{Address: account})
	if err != nil {
		return database.SignedTx{}, err
	}

	ksAccountJson, err := ioutil.ReadFile(ksAccount.URL.Path)
	if err != nil {
		return database.SignedTx{}, err
	}

	key, err := keystore.DecryptKey(ksAccountJson, pwd)
	if err != nil {
		return database.SignedTx{}, err
	}

	signedTx, err := SignTx(tx, key.PrivateKey)
	if err != nil {
		return database.SignedTx{}, err
	}

	return signedTx, nil
}

func SignTx(tx database.Tx, privKey *ecdsa.PrivateKey) (database.SignedTx, error) {
	rawTx, err := tx.Encode()
	if err != nil {
		return database.SignedTx{}, err
	}

	sig, err := Sign(rawTx, privKey)
	if err != nil {
		return database.SignedTx{}, err
	}

	return database.NewSignedTx(tx, sig), nil
}

func Sign(msg []byte, privKey *ecdsa.PrivateKey) ([]byte, error) {
	// Hash the message to 32 bytes
	msgHash := crypto.Keccak256(msg)

	// Sign message using the private key
	sig, err := crypto.Sign(msgHash, privKey)
	if err != nil {
		return nil, err
	}

	// Verify the length
	if len(sig) != crypto.SignatureLength {
		return nil, fmt.Errorf("wrong size for signature: got %d, want %d", len(sig), crypto.SignatureLength)
	}

	return sig, nil
}

func Verify(msg, sig []byte) (*ecdsa.PublicKey, error) {
	msgHash := crypto.Keccak256(msg)

	recoveredPubKey, err := crypto.SigToPub(msgHash, sig)
	if err != nil {
		return nil, fmt.Errorf("unable to verify message signature. %s", err.Error())
	}

	return recoveredPubKey, err
}