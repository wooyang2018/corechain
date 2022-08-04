package bridge

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/wooyang2018/corechain/contract/base"
	"github.com/wooyang2018/corechain/crypto/core/hash"
	"github.com/wooyang2018/corechain/protos"
	"google.golang.org/protobuf/proto"
)

type contractManager struct {
	xbridge      *XBridge
	codeProvider ContractCodeProvider
}

// DeployContract deploy contract and initialize contract
func (c *contractManager) DeployContract(kctx base.KContext) (*base.Response, base.Limits, error) {
	args := kctx.Args()
	state := kctx
	name := args["contract_name"]
	if name == nil {
		return nil, base.Limits{}, errors.New("bad contract name")
	}
	contractName := string(name)
	_, err := c.codeProvider.GetContractCodeDesc(contractName)
	if err == nil {
		return nil, base.Limits{}, fmt.Errorf("contract %s already exists", contractName)
	}

	code := args["contract_code"]
	if code == nil {
		return nil, base.Limits{}, errors.New("missing contract code")
	}
	initArgsBuf := args["init_args"]
	if initArgsBuf == nil {
		return nil, base.Limits{}, errors.New("missing args field in args")
	}
	var initArgs map[string][]byte
	err = json.Unmarshal(initArgsBuf, &initArgs)
	if err != nil {
		return nil, base.Limits{}, err
	}

	descbuf := args["contract_desc"]
	var desc protos.WasmCodeDesc
	err = proto.Unmarshal(descbuf, &desc)
	if err != nil {
		return nil, base.Limits{}, err
	}
	desc.Digest = hash.DoubleSha256(code)
	descbuf, _ = proto.Marshal(&desc)

	if err := state.Put("contract", base.ContractCodeDescKey(contractName), descbuf); err != nil {
		return nil, base.Limits{}, err
	}
	if err := state.Put("contract", base.ContractCodeKey(contractName), code); err != nil {
		return nil, base.Limits{}, err

	}

	if desc.ContractType == string(TypeEvm) {
		abiBuf := args["contract_abi"]
		if err := state.Put("contract", base.ContractAbiKey(contractName), abiBuf); err != nil {
			return nil, base.Limits{}, err
		}
	}

	contractType, err := getContractType(&desc)
	if err != nil {
		return nil, base.Limits{}, err
	}
	creator := c.xbridge.getCreator(contractType)
	if creator == nil {
		return nil, base.Limits{}, fmt.Errorf("contract type %s not found", contractType)
	}
	cp := newCodeProviderWithCache(state)
	instance, err := creator.CreateInstance(&Context{
		State:          state,
		ContractName:   contractName,
		Method:         "initialize",
		ResourceLimits: kctx.ResourceLimit(),
	}, cp)
	if err != nil {
		creator.RemoveCache(contractName)
		// log.Error("create contract instance error when deploy contract", "error", err, "contract", contractName)
		return nil, base.Limits{}, err
	}
	instance.Release()

	initConfig := base.ContextConfig{
		ResourceLimits:        kctx.ResourceLimit(),
		State:                 kctx,
		Initiator:             kctx.Initiator(),
		AuthRequire:           kctx.AuthRequire(),
		ContractName:          contractName,
		CanInitialize:         true,
		ContractCodeFromCache: true,
	}
	initConfig.ContractName = contractName
	initConfig.CanInitialize = true
	initConfig.ContractCodeFromCache = true
	initConfig.State = kctx
	out, resourceUsed, err := c.initContract(contractType, &initConfig, initArgs)
	if err != nil {
		if _, ok := err.(*ContractError); !ok {
			creator.RemoveCache(contractName)
		}
		// log.Error("call contract initialize method error", "error", err, "contract", contractName)
		return nil, base.Limits{}, err
	}
	return out, resourceUsed, nil
}

func (v *contractManager) initContract(tp ContractType, contextConfig *base.ContextConfig, args map[string][]byte) (*base.Response, base.Limits, error) {
	ctx, err := v.xbridge.NewContext(contextConfig)
	if err != nil {
		return nil, base.Limits{}, err
	}
	out, err := ctx.Invoke("initialize", args)
	if err != nil {
		return nil, base.Limits{}, err
	}
	return out, ctx.ResourceUsed(), nil
}

// UpgradeContract deploy contract and initialize contract
func (c *contractManager) UpgradeContract(kctx base.KContext) (*base.Response, base.Limits, error) {
	args := kctx.Args()
	if !c.xbridge.config.EnableUpgrade {
		return nil, base.Limits{}, errors.New("contract upgrade disabled")
	}

	name := args["contract_name"]
	if name == nil {
		return nil, base.Limits{}, errors.New("bad contract name")
	}
	contractName := string(name)
	desc, err := c.codeProvider.GetContractCodeDesc(contractName)
	if err != nil {
		return nil, base.Limits{}, fmt.Errorf("contract %s not exists", contractName)
	}

	code := args["contract_code"]
	if code == nil {
		return nil, base.Limits{}, errors.New("missing contract code")
	}
	desc.Digest = hash.DoubleSha256(code)
	descbuf, _ := proto.Marshal(desc)

	store := kctx
	store.Put("contract", base.ContractCodeDescKey(contractName), descbuf)
	store.Put("contract", base.ContractCodeKey(contractName), code)

	cp := newCodeProviderWithCache(store)

	contractType, err := getContractType(desc)
	if err != nil {
		return nil, base.Limits{}, err
	}
	creator := c.xbridge.getCreator(contractType)
	if creator == nil {
		return nil, base.Limits{}, fmt.Errorf("contract type %s not found", contractType)
	}
	instance, err := creator.CreateInstance(&Context{
		ContractName:   contractName,
		ResourceLimits: base.MaxLimits,
	}, cp)
	if err != nil {
		// log.Error("create contract instance error when upgrade contract", "error", err, "contract", contractName)
		return nil, base.Limits{}, err
	}
	instance.Release()

	return &base.Response{
			Status: 200,
			Body:   []byte("upgrade success"),
		}, base.Limits{
			Disk: base.ModelCacheDiskUsed(store),
		}, nil
}

func getContractType(desc *protos.WasmCodeDesc) (ContractType, error) {
	switch desc.ContractType {
	case "", "wasm":
		return TypeWasm, nil
	case "native":
		return TypeNative, nil
	case "evm":
		return TypeEvm, nil
	case "xkernel":
		return TypeKernel, nil
	default:
		return "", fmt.Errorf("unknown contract type:%s", desc.ContractType)
	}
}
