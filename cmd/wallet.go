package cmd

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-address"
	cli2 "github.com/filecoin-project/lotus/cli"
	"github.com/urfave/cli/v2"
	"github.com/yg-xt/lotus-sign/service/mpool"
	"github.com/yg-xt/lotus-sign/service/wallet"
	"os"
	"strings"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/lib/tablewriter"
)

var WalletCmd = &cli.Command{
	Name:  "wallet",
	Usage: "Manage wallet",
	Subcommands: []*cli.Command{
		walletNew,
		walletList,
		walletExport,
		walletImport,
		walletSign,
		walletDelete,
	},
}

var walletNew = &cli.Command{
	Name:      "new",
	Usage:     "Generate a new key of the given type",
	ArgsUsage: "[bls|secp256k1 (default secp256k1)]",
	Action: func(cctx *cli.Context) error {
		err := wallet.InitRepo(WalletRepo)
		if err != nil {
			fmt.Println(err)
			return err
		}
		walletnew, err := wallet.NewWallet(WalletRepo)
		ctx := cli2.ReqContext(cctx)

		afmt := cli2.NewAppFmt(cctx.App)

		t := cctx.Args().First()
		if t == "" {
			t = "secp256k1"
		}

		nk, err := walletnew.WalletNew(ctx, types.KeyType(t))
		if err != nil {
			return err
		}

		afmt.Println(nk.String())

		return nil
	},
}

var walletList = &cli.Command{
	Name:  "list",
	Usage: "List wallet address",

	Action: func(cctx *cli.Context) error {
		err := wallet.InitRepo(WalletRepo)
		if err != nil {
			fmt.Println(err)
			return err
		}
		walletnew, err := wallet.NewWallet(WalletRepo)
		if err != nil {
			return err
		}
		api, closer, err := cli2.GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := cli2.ReqContext(cctx)

		afmt := cli2.NewAppFmt(cctx.App)

		addrs, err := walletnew.WalletList(ctx)
		if err != nil {
			return err
		}

		// Assume an error means no default key is set
		def, _ := walletnew.WalletDefaultAddress(ctx)

		tw := tablewriter.New(
			tablewriter.Col("Address"),
			tablewriter.Col("ID"),
			tablewriter.Col("Balance"),
			tablewriter.Col("Market(Avail)"),
			tablewriter.Col("Market(Locked)"),
			tablewriter.Col("Nonce"),
			tablewriter.Col("Default"),
			tablewriter.NewLineCol("Error"))

		for _, addr := range addrs {
			if cctx.Bool("addr-only") {
				afmt.Println(addr.String())
			} else {
				a, err := api.StateGetActor(ctx, addr, types.EmptyTSK)
				if err != nil {
					if !strings.Contains(err.Error(), "actor not found") {
						tw.Write(map[string]interface{}{
							"Address": addr,
							"Error":   err,
						})
						continue
					}

					a = &types.Actor{
						Balance: big.Zero(),
					}
				}

				row := map[string]interface{}{
					"Address": addr,
					"Balance": types.FIL(a.Balance),
					"Nonce":   a.Nonce,
				}
				if addr == def {
					row["Default"] = "X"
				}

				if cctx.Bool("id") {
					id, err := api.StateLookupID(ctx, addr, types.EmptyTSK)
					if err != nil {
						row["ID"] = "n/a"
					} else {
						row["ID"] = id
					}
				}

				tw.Write(row)
			}
		}

		if !cctx.Bool("addr-only") {
			return tw.Flush(os.Stdout)
		}

		return nil
	},
}

var walletImport = &cli.Command{
	Name:      "import",
	Usage:     "import keys",
	ArgsUsage: "[<path> (optional, will read from stdin if omitted)]",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "format",
			Usage: "specify input format for key",
			Value: "hex-lotus",
		},
	},
	Action: func(cctx *cli.Context) error {
		err := wallet.InitRepo(WalletRepo)
		if err != nil {
			fmt.Println(err)
			return err
		}
		walletnew, err := wallet.NewWallet(WalletRepo)
		ctx := cli2.ReqContext(cctx)

		var inpdata []byte
		if !cctx.Args().Present() || cctx.Args().First() == "-" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter private key: ")
			indata, err := reader.ReadBytes('\n')
			if err != nil {
				return err
			}
			inpdata = indata

		} else {
			fdata, err := os.ReadFile(cctx.Args().First())
			if err != nil {
				return err
			}
			inpdata = fdata
		}

		var ki types.KeyInfo
		switch cctx.String("format") {
		case "hex-lotus":
			data, err := hex.DecodeString(strings.TrimSpace(string(inpdata)))
			if err != nil {
				return err
			}

			if err := json.Unmarshal(data, &ki); err != nil {
				return err
			}
		case "json-lotus":
			if err := json.Unmarshal(inpdata, &ki); err != nil {
				return err
			}
		case "gfc-json":
			var f struct {
				KeyInfo []struct {
					PrivateKey []byte
					SigType    int
				}
			}
			if err := json.Unmarshal(inpdata, &f); err != nil {
				return xerrors.Errorf("failed to parse go-filecoin key: %s", err)
			}

			gk := f.KeyInfo[0]
			ki.PrivateKey = gk.PrivateKey
			switch gk.SigType {
			case 1:
				ki.Type = types.KTSecp256k1
			case 2:
				ki.Type = types.KTBLS
			default:
				return fmt.Errorf("unrecognized key type: %d", gk.SigType)
			}
		default:
			return fmt.Errorf("unrecognized format: %s", cctx.String("format"))
		}

		addr, err := walletnew.WalletImport(ctx, &ki)
		if err != nil {
			return err
		}

		fmt.Printf("imported key %s successfully!\n", addr)
		return nil
	},
}

var walletExport = &cli.Command{
	Name:      "export",
	Usage:     "export keys",
	ArgsUsage: "[address]",
	Action: func(cctx *cli.Context) error {
		walletnew, err := wallet.NewWallet(WalletRepo)
		ctx := cli2.ReqContext(cctx)
		afmt := cli2.NewAppFmt(cctx.App)
		addr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}

		ki, err := walletnew.WalletExport(ctx, addr)
		if err != nil {
			return err
		}

		b, err := json.Marshal(ki)
		if err != nil {
			return err
		}

		afmt.Println(hex.EncodeToString(b))
		return nil
	},
}

var walletSign = &cli.Command{
	Name:      "sign",
	Usage:     "sign a message",
	ArgsUsage: "<signing address> <hexMessage>",
	Action: func(cctx *cli.Context) error {
		walletnew, err := wallet.NewWallet(WalletRepo)
		if err != nil {
			return err
		}
		ctx := cli2.ReqContext(cctx)

		afmt := cli2.NewAppFmt(cctx.App)

		if cctx.NArg() != 2 {
			return cli2.IncorrectNumArgs(cctx)
		}

		addr, err := address.NewFromString(cctx.Args().First())

		if err != nil {
			return err
		}

		msg, err := hex.DecodeString(cctx.Args().Get(1))

		if err != nil {
			return err
		}
		var smg = &types.Message{}
		err = json.Unmarshal(msg, &smg)
		if err != nil {
			return err
		}
		sig, err := walletnew.WalletSign(ctx, addr, msg)

		if err != nil {
			return err
		}

		sigBytes := append([]byte{byte(sig.Type)}, sig.Data...)

		sb, err := mpool.SigningBytes(smg, smg.From.Protocol())
		if err != nil {
			return err
		}

		_, err = walletnew.WalletVerify(ctx, addr, sb, sig)
		if err != nil {
			return err
		}
		afmt.Println(hex.EncodeToString(sigBytes))
		return nil
	},
}

var walletDelete = &cli.Command{
	Name:      "delete",
	Usage:     "Soft delete an address from the wallet - hard deletion needed for permanent removal",
	ArgsUsage: "<address> ",
	Action: func(cctx *cli.Context) error {
		walletnew, err := wallet.NewWallet(WalletRepo)
		if err != nil {
			return err
		}
		ctx := cli2.ReqContext(cctx)
		if cctx.NArg() != 1 {
			return cli2.IncorrectNumArgs(cctx)
		}

		addr, err := address.NewFromString(cctx.Args().First())
		if err != nil {
			return err
		}
		fmt.Printf("Delete the wallet address with caution. The secret key cannot be recovered. : %s\n", addr.String())
		return walletnew.WalletDelete(ctx, addr)
	},
}
