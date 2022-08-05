package native

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/wooyang2018/corechain/contract/base"
	_ "github.com/wooyang2018/corechain/contract/kernel"
	"github.com/wooyang2018/corechain/contract/mock"
	"github.com/wooyang2018/corechain/logger"
	mockConf "github.com/wooyang2018/corechain/mock/config"
)

const (
	RUNTIME_DOCKER = "docker"
	RUNTIME_HOST   = "host"
	IMAGE_NAME     = "alpine"
)

func compile(th *mock.TestHelper, runtime string) ([]byte, error) {
	target := filepath.Join(th.Basedir(), "counter.bin")
	cmd := exec.Command("go", "build", "-o", target)
	if runtime == RUNTIME_DOCKER {
		cmd.Env = append(os.Environ(), []string{"GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0"}...)
	}
	cmd.Dir = "testdata"
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s:%s", err, out)
	}
	bin, err := ioutil.ReadFile(target)
	if err != nil {
		return nil, err
	}
	return bin, nil
}

func TestNative(t *testing.T) {
	mockConf.InitFakeLogger()
	l, _ := logger.NewLogger("", "native_test")
	var contractConfig = &base.ContractConfig{
		EnableUpgrade: true,
		Xkernel: base.XkernelConfig{
			Enable: true,
			Driver: "default",
		},
		Native: base.NativeConfig{
			Enable: true,
			Driver: "native",
			Docker: base.NativeDockerConfig{
				Enable:    true,
				ImageName: IMAGE_NAME,
			},
		},
		LogDriver: l,
	}

	runtimes := []string{RUNTIME_HOST, RUNTIME_DOCKER}

	for _, runtime := range runtimes {
		if runtime == RUNTIME_DOCKER {
			_, err := exec.Command("docker", "info").CombinedOutput()
			if err != nil {
				t.Skip("docker not available")
			}

			t.Log("pulling image......")
			pullResp, errPull := exec.Command("docker", "pull", IMAGE_NAME).CombinedOutput()
			if errPull != nil {
				t.Error(err)
				continue
				t.Log(string(pullResp))
			}
			contractConfig.Native.Docker.Enable = true
		} else {
			contractConfig.Native.Docker.Enable = false
		}
		t.Run("TestNativeDeploy_"+runtime, func(t *testing.T) {
			th := mock.NewTestHelper(contractConfig)
			defer th.Close()

			bin, err := compile(th, runtime)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := th.Deploy("native", "go", "counter", bin, map[string][]byte{
				"creator": []byte("icexin"),
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%#v", resp)
		})
		t.Run("TestNativeInvoke_"+runtime, func(t *testing.T) {
			th := mock.NewTestHelper(contractConfig)
			defer th.Close()

			bin, err := compile(th, runtime)
			if err != nil {
				t.Fatal(err)
			}

			_, err = th.Deploy("native", "go", "counter", bin, map[string][]byte{
				"creator": []byte("icexin"),
			})
			if err != nil {
				t.Fatal(err)
			}

			resp, err := th.Invoke("native", "counter", "increase", map[string][]byte{
				"key": []byte("k1"),
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("body:%s", resp.Body)
		})

		t.Run("TestNativeUpgrade_"+runtime, func(t *testing.T) {
			th := mock.NewTestHelper(contractConfig)
			defer th.Close()

			bin, err := compile(th, runtime)
			if err != nil {
				t.Fatal(err)
			}

			_, err = th.Deploy("native", "go", "counter", bin, map[string][]byte{
				"creator": []byte("icexin"),
			})
			if err != nil {
				t.Fatal(err)
			}

			err = th.Upgrade("counter", bin)
			if err != nil {
				t.Fatal(err)
			}
		})

	}
}
