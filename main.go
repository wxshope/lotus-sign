package main

import (
	"github.com/filecoin-project/lotus/lib/lotuslog"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"github.com/yg-xt/lotus-sign/cmd"
	"os"
)

var log = logging.Logger("main")

func main() {
	lotuslog.SetupLogLevels()

	app := cli.App{
		Commands: []*cli.Command{
			cmd.ActorWithdrawCmd,
			cmd.SendCmd,
			cmd.WalletCmd,
			cmd.ActorSetOwnerCmd,
			cmd.MpoolReplaceCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Warnf("%+v", err)
		return
	}
}
