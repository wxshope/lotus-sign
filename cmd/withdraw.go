package cmd

import (
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	"github.com/urfave/cli/v2"
	"github.com/yg-xt/lotus-sign/service/mpool"
	"github.com/yg-xt/lotus-sign/service/wallet"
	"golang.org/x/xerrors"
	"time"
)
var ActorWithdrawCmd = &cli.Command{
	Name:      "withdraw",
	Usage:     "withdraw available balance to beneficiary",
	ArgsUsage: "[amount (FIL)]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "actor",
			Usage: "矿工地址 miner actor",
		},
		&cli.StringFlag{
			Name:  "from",
			Usage: "矿工 owner钱包地址",
		},
	},
	Action: func(cctx *cli.Context) error {
		walletnew, err := wallet.NewWallet(wallet.LotusRepo)
		if err != nil {
			return err
		}
		var miner address.Address
		if act := cctx.String("actor"); act != "" {
			var err error
			miner, err = address.NewFromString(act)
			if err != nil {
				return fmt.Errorf("parsing address %s: %w", act, err)
			}
		}
		var newowner address.Address
		if act := cctx.String("from"); act != "" {
			var err error
			newowner, err = address.NewFromString(act)
			if err != nil {
				return fmt.Errorf("parsing address %s: %w", act, err)
			}
		}

		NodeAPI, acloser, err := lcli.GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer acloser()

		ctx := lcli.ReqContext(cctx)

		if miner.Empty() {
			minerApi, closer, err := lcli.GetStorageMinerAPI(cctx)
			if err != nil {
				return err
			}
			defer closer()

			miner, err = minerApi.ActorAddress(ctx)
			if err != nil {
				return err
			}
		}

		mi, err := NodeAPI.StateMinerInfo(ctx, miner, types.EmptyTSK)
		if err != nil {
			return err
		}

		available, err := NodeAPI.StateMinerAvailableBalance(ctx, miner, types.EmptyTSK)
		if err != nil {
			return err
		}

		amount := available
		if cctx.Args().Present() {
			f, err := types.ParseFIL(cctx.Args().First())
			if err != nil {
				return xerrors.Errorf("parsing 'amount' argument: %w", err)
			}

			amount = abi.TokenAmount(f)

			if amount.GreaterThan(available) {
				return xerrors.Errorf("can't withdraw more funds than available; requested: %s; available: %s", types.FIL(amount), types.FIL(available))
			}
		}

		params, err := actors.SerializeParams(&miner2.WithdrawBalanceParams{
			AmountRequested: amount, // Default to attempting to withdraw all the extra funds in the miner actor
		})
		if err != nil {
			return err
		}

		nonce, err := NodeAPI.MpoolGetNonce(ctx, mi.Owner)
		if err != nil {
			return err
		}
		owner, err := NodeAPI.StateAccountKey(ctx, mi.Owner, types.EmptyTSK)
		if err != nil {
			fmt.Printf("%s error getting account key: %s\n", mi.Owner, err)
			return err
		}

		if newowner != owner {
			fmt.Println("owner钱包地址正确")
			err := xerrors.New("owner钱包地址正确")
			return err
		}
		var msg = &types.Message{
			To:         miner,
			From:       owner,
			Nonce:      nonce,
			Value:      types.NewInt(0),
			Method:     builtin.MethodsMiner.WithdrawBalance,
			Params:     params,
			GasPremium: types.NewInt(0),
			GasFeeCap:  types.NewInt(0),
			GasLimit:   0,
		}

		smg, err := mpool.GasLimit(ctx, NodeAPI, msg)
		if err != nil {
			return err
		}
		sb, err := mpool.SigningBytes(msg, smg.From.Protocol())
		if err != nil {
			return err
		}

		sig, err := walletnew.WalletSign(ctx, msg.From, sb)
		if err != nil {
			return err
		}

		smsg := &types.SignedMessage{
			Message:   *msg,
			Signature: *sig,
		}
		_, err = walletnew.WalletVerify(ctx, msg.From, sb, sig)
		if err != nil {
			err = xerrors.Errorf("签名验证失败 %s\n", err)
			return err
		}

		sm, err := NodeAPI.MpoolPush(ctx, smsg)
		if err != nil {
			return xerrors.Errorf("failed to push new message to mempool: %w", err)
		}
		fmt.Println("message successfully!", sm.String())
		// wait for it to get mined into a block
		for {
			msg, err1 := NodeAPI.StateReplay(ctx, types.TipSetKey{}, sm)
			if err1 != nil {
				//log.Error("获取消息失败：", err1)
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