package mock

import (
	"fmt"
	"math"
	"math/rand"
	"path/filepath"
	"strconv"
	"time"

	xconf "github.com/wooyang2018/corechain/common/config"
	"github.com/wooyang2018/corechain/common/utils"
)

var dir = utils.GetCurFileDir()

func GetEnvConfFilePath() string {
	return filepath.Join(dir, "../conf/env.yaml")
}

func GetEnvDataDirPath() string {
	return filepath.Join(dir, "../data/chains")
}

func GetGenesisConfFilePath(name string) string {
	return filepath.Join(dir, fmt.Sprintf("../data/genesis/%s.json", name))
}

func GetTempDirPath() string {
	rand.Seed(time.Now().UnixNano())
	return filepath.Join("temp", strconv.Itoa(rand.Intn(math.MaxInt)))
}

func GetServerConfFilePath() string {
	return filepath.Join(dir, "../conf/server.yaml")
}

func GetMockEnvConf(paths ...string) (*xconf.EnvConf, error) {
	path := "../conf/env.yaml"
	if len(paths) > 0 {
		path = paths[0]
	}

	econfPath := filepath.Join(dir, path)
	econf, err := xconf.LoadEnvConf(econfPath)
	if err != nil {
		return nil, err
	}

	return econf, nil
}
