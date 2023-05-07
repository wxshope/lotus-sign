package cmd

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/types"
	cli2 "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"
	"time"
)

var MpoolReplaceCmd = &cli.Command{
	Name:  "replace_sign",
	Usage: "replace a message in the mempool",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "msg",
			Usage: "sign msg",
		},
	},
	Action: func(cctx *cli.Context) error {
		NodeAPI, closer, err := cli2.GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()

		ctx := cli2.ReqContext(cctx)
		msg := cctx.String("msg")
		buf, err := hex.DecodeString(msg)
		if err != nil {
			panic(err)
		}

		sig := new(types.SignedMessage)
		if err := sig.UnmarshalCBOR(bytes.NewReader(buf)); err != nil {
			panic(err)
		}
		sm, err := NodeAPI.MpoolPush(ctx, sig)
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
