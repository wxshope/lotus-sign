package cmd

import (
	"encoding/hex"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/build"
	cli2 "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
	"github.com/yg-xt/lotus-sign/service/mpool"
	"github.com/yg-xt/lotus-sign/service/wallet"
	"golang.org/x/xerrors"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/actors/builtin"
	"github.com/filecoin-project/lotus/chain/types"
)

var SendCmd = &cli.Command{
	Name:      "send",
	Usage:     "Send funds between accounts",
	ArgsUsage: "[targetAddress] [amount]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from",
			Usage: "optionally specify the account to send funds from",
		},

		&cli.Uint64Flag{
			Name:  "method",
			Usage: "specify method to invoke",
			Value: uint64(builtin.MethodSend),
		},
		&cli.StringFlag{
			Name:  "params-json",
			Usage: "specify invocation parameters in json",
		},
		&cli.StringFlag{
			Name:  "params-hex",
			Usage: "specify invocation parameters in hex",
		},
	},
	Action: func(cctx *cli.Context) error {
		walletnew, err := wallet.NewWallet(WalletRepo)
		if err != nil {
			return err
		}
		if cctx.IsSet("force") {
			fmt.Println("'force' flag is deprecated, use global flag 'force-send'")
		}

		if cctx.NArg() != 2 {
			return cli2.IncorrectNumArgs(cctx)
		}

		NodeAPI, acloser, err := cli2.GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer acloser() //nolint:errcheck

		ctx := cli2.ReqContext(cctx)
		var params cli2.SendParams

		params.To, err = address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return cli2.ShowHelp(cctx, fmt.Errorf("failed to parse target address: %w", err))
		}

		val, err := types.ParseFIL(cctx.Args().Get(1))
		if err != nil {
			return cli2.ShowHelp(cctx, fmt.Errorf("failed to parse amount: %w", err))
		}
		params.Val = abi.TokenAmount(val)

		if from := cctx.String("from"); from != "" {
			addr, err := address.NewFromString(from)
			if err != nil {
				return err
			}

			params.From = addr
		}

		if cctx.IsSet("params-hex") {
			decparams, err := hex.DecodeString(cctx.String("params-hex"))
			if err != nil {
				return fmt.Errorf("failed to decode hex params: %w", err)
			}
			params.Params = decparams
		}
		if cctx.IsSet("params-json") {
			if params.Params != nil {
				return fmt.Errorf("can only specify one of 'params-json' and 'params-hex'")
			}
			act, err := NodeAPI.StateGetActor(ctx, params.To, types.EmptyTSK)
			if err != nil {
				return err
			}
			decparams, err := wallet.DecodeTypedParamsFromJSON(act, params.Method, cctx.String("params-json"))
			if err != nil {
				return fmt.Errorf("failed to decode json params: %w", err)
			}
			params.Params = decparams
		}

		nonce, err := NodeAPI.MpoolGetNonce(ctx, params.From)
		var msg = &types.Message{
			To:         params.To,
			From:       params.From,
			Nonce:      nonce,
			Value:      params.Val,
			GasLimit:   0,
			GasPremium: types.NewInt(0),
			GasFeeCap:  types.NewInt(0),
			Method:     builtin.MethodSend,
		}
		smg, err := mpool.GasLimit(ctx, NodeAPI, msg)
		if err != nil {
			return err
		}
		sb, err := mpool.SigningBytes(smg, msg.From.Protocol())
		if err != nil {
			return err
		}
		sig, err := walletnew.WalletSign(ctx, msg.From, sb)
		if err != nil {
			return err
		}
		var msig = &types.SignedMessage{
			Message:   *msg,
			Signature: *sig,
		}
		_, err = walletnew.WalletVerify(ctx, msg.From, sb, sig)
		if err != nil {
			err = xerrors.Errorf("签名验证失败 %s\n", err)
			return err
		}

		sm, err := NodeAPI.MpoolPush(ctx, msig)
		if err != nil {
			return err
		}
		fmt.Println("message successfully!", sm.String())

		for {
			msg, err := NodeAPI.StateReplay(ctx, types.TipSetKey{}, sm)
			if err != nil {
				time.Sleep(time.Second * 2)
				continue
			}

			wait, err := NodeAPI.StateWaitMsg(ctx, sm, build.MessageConfidence)
			if err != nil {
				return err
			}
			// check it executed successfully
			if wait.Receipt.ExitCode.IsError() {
				fmt.Println("send msg failed!")
				return err
			}

			fmt.Println("message succeeded!", msg.MsgCid.String())
			break
		}

		return nil
	},
}
