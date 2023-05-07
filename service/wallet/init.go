package wallet

import (
	logging "github.com/ipfs/go-log/v2"
)

const (
	KNamePrefix  = "wallet-"
	KDefault     = "default"
	fsKeystore   = "keystore"
	KTrashPrefix = "trash-"
)

var log = logging.Logger("wallet")
