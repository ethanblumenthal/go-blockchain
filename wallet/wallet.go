package wallet

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
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