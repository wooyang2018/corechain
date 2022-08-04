package bridge

import (
	"errors"
	"fmt"

	"github.com/wooyang2018/corechain/contract/base"
)

const (
	initMethod = "initialize"
)

// ContractError indicates the error of the contract running result
type ContractError struct {
	Status  int
	Message string
}

// Error implements error interface
func (c *ContractError) Error() string {
	return fmt.Sprintf("contract error status:%d message:%s", c.Status, c.Message)
}

// VMContextImpl 为vm.Context的实现，
// 它组合了合约内核态数据(ctx)以及用户态的虚拟机数据(instance)
type VMContextImpl struct {
	ctx      *Context
	instance Instance
	release  func()
}

func (v *VMContextImpl) Invoke(method string, args map[string][]byte) (*base.Response, error) {
	if !v.ctx.CanInitialize && method == initMethod {
		return nil, errors.New("invalid contract method " + method)
	}

	v.ctx.Method = method
	v.ctx.Args = args
	err := v.instance.Exec()
	if err != nil {
		return nil, err
	}

	if v.ctx.ResourceUsed().Exceed(v.ctx.ResourceLimits) {
		return nil, errors.New("resource exceeds")
	}

	if v.ctx.Output == nil {
		return nil, &ContractError{
			Status:  500,
			Message: "internal error",
		}
	}

	return &base.Response{
		Status:  int(v.ctx.Output.GetStatus()),
		Message: v.ctx.Output.GetMessage(),
		Body:    v.ctx.Output.GetBody(),
	}, nil
}

func (v *VMContextImpl) ResourceUsed() base.Limits {
	return v.ctx.ResourceUsed()
}

func (v *VMContextImpl) Release() error {
	// release the context of instance
	v.instance.Release()
	v.release()
	return nil
}
