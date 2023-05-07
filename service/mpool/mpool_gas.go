package mpool

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/types/ethtypes"
	"golang.org/x/xerrors"
)

func SigningBytes(msg *types.Message, sigType address.Protocol) ([]byte, error) {
	if sigType == address.Delegated {
		txArgs, err := ethtypes.EthTxArgsFromUnsignedEthMessage(msg)
		if err != nil {
			return nil, xerrors.Errorf("failed to reconstruct eth transaction: %w", err)
		}
		rlpEncodedMsg, err := txArgs.ToRlpUnsignedMsg()
		if err != nil {
			return nil, xerrors.Errorf("failed to repack eth rlp message: %w", err)
		}
		return rlpEncodedMsg, nil
	}

	return msg.Cid().Bytes(), nil
}

func GasLimit(ctx context.Context, nodeAPI v0api.FullNode, params *types.Message) (msg *types.Message, err error) {

	if params.GasLimit == 0 {
		gasLimit, err := nodeAPI.GasEstimateGasLimit(ctx, params, types.EmptyTSK)
		if err != nil {
			return nil, err
		}
		params.GasLimit = int64(float64(gasLimit) * GasLimitOverestimation)
	}

	if params.GasPremium == types.EmptyInt || types.BigCmp(params.GasPremium, types.NewInt(0)) == 0 {
		gasPremium, err := nodeAPI.GasEstimateGasPremium(ctx, 10, params.From, params.GasLimit, types.EmptyTSK)
		if err != nil {
			return nil, xerrors.Errorf("estimating gas price: %w", err)
		}
		params.GasPremium = gasPremium
	}

	if params.GasFeeCap == types.EmptyInt || types.BigCmp(params.GasFeeCap, types.NewInt(0)) == 0 {
		feeCap, err := nodeAPI.GasEstimateFeeCap(ctx, params, 20, types.EmptyTSK)
		if err != nil {
			return nil, xerrors.Errorf("estimating fee cap: %w", err)
		}
		params.GasFeeCap = feeCap
	}
	CapGasFee(params)
	msg = params
	return
}

func CapGasFee(msg *types.Message) {
	var maxFee abi.TokenAmount

	if maxFee.Int == nil || maxFee.Equals(big.Zero()) {
		maxFee, _ = big.FromString("10000000000000000000")
	}

	gl := types.NewInt(uint64(msg.GasLimit))
	totalFee := types.BigMul(msg.GasFeeCap, gl)

	if totalFee.LessThanEqual(maxFee) {
		msg.GasPremium = big.Min(msg.GasFeeCap, msg.GasPremium) // cap premium at FeeCap
		return
	}

	msg.GasFeeCap = big.Div(maxFee, gl)
	msg.GasPremium = big.Min(msg.GasFeeCap, msg.GasPremium) // cap premium at FeeCap
}
