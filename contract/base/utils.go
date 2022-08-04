package base

import (
	"fmt"
	"regexp"
)

var (
	contractNameRegex = regexp.MustCompile("^[a-zA-Z_]{1}[0-9a-zA-Z_.]+[0-9a-zA-Z_]$")
)

const (
	contractNameMaxSize = 16
	contractNameMinSize = 4
)

// ValidContractName return error when contractName is not a valid contract name.
func ValidContractName(contractName string) error {
	// param absence check
	// contract naming rule check
	contractSize := len(contractName)
	contractMaxSize := contractNameMaxSize
	contractMinSize := contractNameMinSize
	if contractSize > contractMaxSize || contractSize < contractMinSize {
		return fmt.Errorf("contract name length expect [%d~%d], actual: %d", contractMinSize, contractMaxSize, contractSize)
	}
	if !contractNameRegex.MatchString(contractName) {
		return fmt.Errorf("contract name does not fit the rule of contract name")
	}
	return nil
}

func ModelCacheDiskUsed(store KContext) int64 {
	size := int64(0)
	wset := store.RWSet().WSet
	for _, w := range wset {
		size += int64(len(w.GetKey()))
		size += int64(len(w.GetValue()))
	}
	return size
}

func ContractCodeDescKey(contractName string) []byte {
	return []byte(contractName + "." + "desc")
}

func ContractCodeKey(contractName string) []byte {
	return []byte(contractName + "." + "code")
}

func ContractAbiKey(contractName string) []byte {
	return []byte(contractName + "." + "abi")
}
