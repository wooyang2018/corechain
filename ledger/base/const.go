package base

const (
	LedgerSubModName = "ledger"
	StateSubModName  = "state"
	// ledger storage dir name
	LedgerStrgDirName = "ledger"
	// state machine storage dir name
	StateStrgDirName = "utxovm"
)

// base definition for KV prefix
const (
	BlocksTablePrefix        = "B"
	UTXOTablePrefix          = "U"
	UnconfirmedTablePrefix   = "N"
	ConfirmedTablePrefix     = "C"
	MetaTablePrefix          = "M"
	PendingBlocksTablePrefix = "PB"
	ExtUtxoDelTablePrefix    = "ZD"
	ExtUtxoTablePrefix       = "ZU"
	BlockHeightPrefix        = "ZH"
	BranchInfoPrefix         = "ZI"
)
