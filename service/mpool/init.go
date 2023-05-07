package mpool

import logging "github.com/ipfs/go-log/v2"

var log = logging.Logger("mpool")

const (
	dsKeyMsgUUIDSet        = "MsgUuidSet"
	GasLimitOverestimation = 1.25
)
