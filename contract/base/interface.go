package base

type KernRegistry interface {
	RegisterKernMethod(contract, method string, handler KernMethod)
	UnregisterKernMethod(ctract, method string)
	// RegisterShortcut 用于contractName缺失的时候选择哪个合约名字和合约方法来执行对应的kernel合约
	RegisterShortcut(oldmethod, contract, method string)
	GetKernMethod(contract, method string) (KernMethod, error)
}

type KernMethod func(ctx KContext) (*Response, error)

type KContext interface {
	// 交易相关数据
	Args() map[string][]byte
	Initiator() string
	Caller() string
	AuthRequire() []string

	// 状态修改接口
	StateSandbox

	AddResourceUsed(delta Limits)
	ResourceLimit() Limits

	Call(module, contract, method string, args map[string][]byte) (*Response, error)

	// 合约异步事件调用
	EmitAsyncTask(event string, args interface{}) error
}

const (
	// StatusOK is used when contract successfully ends.
	StatusOK = 200
	// StatusErrorThreshold is the status dividing line for the normal operation of the contract
	StatusErrorThreshold = 400
	// StatusError is used when contract fails.
	StatusError = 500
)

// VMContext define context interface
type VMContext interface {
	Invoke(method string, args map[string][]byte) (*Response, error)
	ResourceUsed() Limits
	Release() error
}

// Response is the result of the contract run
type Response struct {
	// Status 用于反映合约的运行结果的错误码
	Status int `json:"status"`
	// Message 用于携带一些有用的debug信息
	Message string `json:"message"`
	// Data 字段用于存储合约执行的结果
	Body []byte `json:"body"`
}
