package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wooyang2018/corechain/example/pb"
)

// QueryNominateRecordsCommand query Nominate cmd
type QueryNominateRecordsCommand struct {
	cli  *Cli
	cmd  *cobra.Command
	addr string
}

// NewQueryNominateRecordsCommand new query Nominate cmd
func NewQueryNominateRecordsCommand(cli *Cli) *cobra.Command {
	c := new(QueryNominateRecordsCommand)
	c.cli = cli
	c.cmd = &cobra.Command{
		Use:   "query-nominate-records",
		Short: "Get all records of candidates nominated by an user.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.TODO()
			return c.queryNominateRecords(ctx)
		},
	}
	c.addFlags()
	return c.cmd
}

func (c *QueryNominateRecordsCommand) addFlags() {
	c.cmd.Flags().StringVarP(&c.addr, "addr", "a", "", "user address")
}

func (c *QueryNominateRecordsCommand) queryNominateRecords(ctx context.Context) error {
	client := c.cli.XchainClient()
	request := &pb.DposNominateRecordsRequest{
		Bcname:  c.cli.RootOptions.Name,
		Address: c.addr,
	}

	response, err := client.DposNominateRecords(ctx, request)
	if err != nil {
		return err
	}
	output, err := json.MarshalIndent(response.NominateRecords, "", "  ")
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(string(output))
	return nil
}
