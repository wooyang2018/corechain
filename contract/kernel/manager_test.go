package kernel

import (
	"testing"

	"github.com/wooyang2018/corechain/contract/base"
	mock2 "github.com/wooyang2018/corechain/contract/mock"
	"github.com/wooyang2018/corechain/contract/sandbox"
)

var contractConfig = &base.ContractConfig{
	Xkernel: base.XkernelConfig{
		Enable: true,
		Driver: "default",
	},
	LogDriver: mock2.NewMockLogger(),
}

func TestCreate(t *testing.T) {
	th := mock2.NewTestHelper(contractConfig)
	defer th.Close()
}

func TestCreateSandbox(t *testing.T) {
	th := mock2.NewTestHelper(contractConfig)
	defer th.Close()
	m := th.Manager()

	r := sandbox.NewMemXModel()
	state, err := m.NewStateSandbox(&base.SandboxConfig{
		XMReader: r,
	})
	if err != nil {
		t.Fatal(err)
	}
	state.Put("test", []byte("key"), []byte("value"))
	if string(state.RWSet().WSet[0].Value) != "value" {
		t.Error("unexpected value")
	}
}

func TestInvoke(t *testing.T) {
	th := mock2.NewTestHelper(contractConfig)
	defer th.Close()
	m := th.Manager()

	m.GetKernRegistry().RegisterKernMethod("$hello", "Hi", new(helloContract).Hi)

	resp, err := th.Invoke("xkernel", "$hello", "Hi", map[string][]byte{
		"name": []byte("xuper"),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", resp.Body)
}

type helloContract struct {
}

func (h *helloContract) Hi(ctx base.KContext) (*base.Response, error) {
	name := ctx.Args()["name"]
	ctx.Put("test", []byte("k1"), []byte("v1"))
	return &base.Response{
		Body: []byte("hello " + string(name)),
	}, nil
}
