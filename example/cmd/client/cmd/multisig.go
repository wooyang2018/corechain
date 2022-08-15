package cmd

import "github.com/spf13/cobra"

// MultisigCommand Multisig set command
type MultisigCommand struct {
}

// MultisigData generated multisig data
type MultisigData struct {
	R       []byte   // base random value
	C       []byte   // base public key
	KList   [][]byte // random value list
	PubKeys [][]byte // all public keys
}

// PartialSign partial sign is single Si sign for multisig
type PartialSign struct {
	Si    []byte
	Index int
}

// NewMultisigCommand MultisigCommand init method
func NewMultisigCommand(cli *Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multisig",
		Short: "Operate a command with multisign: check|gen|send|sign|get.",
	}
	cmd.AddCommand(NewMultisigGenCommand(cli))
	cmd.AddCommand(NewGetComplianceCheckSignCommand(cli))
	cmd.AddCommand(NewMultisigCheckCommand(cli))
	cmd.AddCommand(NewMultisigSignCommand(cli))
	cmd.AddCommand(NewMultisigSendCommand(cli))
	return cmd
}

func init() {
	AddCommand(NewMultisigCommand)
}
