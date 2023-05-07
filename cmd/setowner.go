package cmd

import (
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/builtin"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/actors"
	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
	"github.com/yg-xt/lotus-sign/service/mpool"
	"github.com/yg-xt/lotus-sign/service/wallet"
	"golang.org/x/xerrors"
	"time"
)

var ActorSetOwnerCmd = &cli.Command{
	Name:      "set-owner",
	Usage:     "Set owner address (this command should be invoked twice, first with the old owner as the senderAddress, and then with the new owner)",
	ArgsUsage: "[newOwnerAddress senderAddress]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "actor",
			Usage: "specify the address of miner actor",
		},
		&cli.BoolFlag{
			Name:  "really-do-it",
			Usage: "Actually send transaction performing the action",
			Value: false,
		},
	},
	Action: func(cctx *cli.Context) error {
		walletnew, err := wallet.NewWallet(WalletRepo)
		if !cctx.Bool("really-do-it") {
			fmt.Println("Pass --really-do-it to actually execute this action")
			return nil
		}

		if cctx.NArg() != 2 {
			return lcli.IncorrectNumArgs(cctx)
		}

		var maddr address.Address
		if act := cctx.String("actor"); act != "" {
			var err error
			maddr, err = address.NewFromString(act)
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

		na, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		newAddrId, err := NodeAPI.StateLookupID(ctx, na, types.EmptyTSK)
		if err != nil {
			return err
		}

		fa, err := address.NewFromString(cctx.Args().Get(1))
		if err != nil {
			return err
		}

		fromAddrId, err := NodeAPI.StateLookupID(ctx, fa, types.EmptyTSK)
		if err != nil {
			return err
		}

		mi, err := NodeAPI.StateMinerInfo(ctx, maddr, types.EmptyTSK)
		if err != nil {
			return err
		}

		if fromAddrId != mi.Owner && fromAddrId != newAddrId {
			return xerrors.New("from address must either be the old owner or the new owner")
		}

		sp, err := actors.SerializeParams(&newAddrId)
		if err != nil {
			return xerrors.Errorf("serializing params: %w", err)
		}

		FromAddress, err := NodeAPI.StateAccountKey(ctx, fromAddrId, types.EmptyTSK)
		fmt.Println(FromAddress)
		if err != nil {
			return err
		}
		nonce, err := NodeAPI.MpoolGetNonce(ctx, FromAddress)
		if err != nil {
			return err
		}
		var msg = &types.Message{
			From:       FromAddress,
			To:         maddr,
			Nonce:      nonce,
			Value:      big.Zero(),
			GasLimit:   0,
			GasPremium: types.NewInt(0),
			GasFeeCap:  types.NewInt(0),
			Method:     builtin.MethodsMiner.ChangeOwnerAddress,
			Params:     sp,
		}

		smg, err := mpool.GasLimit(ctx, NodeAPI, msg)

		if err != nil {
			fmt.Println(err)
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
