package base

import (
	"errors"
	"time"

	"github.com/wooyang2018/corechain/contract"
)

var (
	EmptyValidors    = errors.New("Current validators is empty.")
	NotValidContract = errors.New("Cannot get valid res with contract.")
	EmptyJustify     = errors.New("Justify is empty.")
	InvalidJustify   = errors.New("Justify structure is invalid.")
)

const (
	SubModName       = "consensus"
	MaxMapSize       = 1000
	StatusOK         = 200
	StatusBadRequest = 400
	StatusErr        = 500
)

func NewContractOKResponse(json []byte) *contract.Response {
	return &contract.Response{
		Status:  StatusOK,
		Message: "success",
		Body:    json,
	}
}

func NewContractErrResponse(msg string) *contract.Response {
	return &contract.Response{
		Status:  StatusErr,
		Message: msg,
	}
}

func NewContractBadResponse(msg string) *contract.Response {
	return &contract.Response{
		Status:  StatusBadRequest,
		Message: msg,
	}
}

// AddressEqual 判断两个validator地址是否相等
func AddressEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, _ := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// CleanProduceMap 删除超出容量的已经落盘的所有key
func CleanProduceMap(isProduce map[int64]bool, period int64) {
	if len(isProduce) <= MaxMapSize {
		return
	}
	t := time.Now().UnixNano() / int64(time.Millisecond)
	key := t / period
	for k, _ := range isProduce {
		if k <= key-int64(MaxMapSize) {
			delete(isProduce, k)
		}
	}
}
