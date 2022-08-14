package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	xconf "github.com/wooyang2018/corechain/common/config"
	"github.com/wooyang2018/corechain/common/utils"
	utils2 "github.com/wooyang2018/corechain/engine/utils"
	"github.com/wooyang2018/corechain/logger"
	_ "github.com/wooyang2018/corechain/storage/leveldb"
)

// CreateChainCommand create chain cmd
type CreateChainCommand struct {
	BaseCmd
	//链名
	Name string
	// 创世块配置文件
	GenesisConf string
	// 环境配置文件
	EnvConf string
}

// NewCreateChainVersion new create chain cmd
func GetCreateChainCommand() *CreateChainCommand {
	c := new(CreateChainCommand)
	c.Cmd = &cobra.Command{
		Use:   "createChain",
		Short: "Create a blockchain.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.CreateChain()
		},
	}

	c.Cmd.Flags().StringVarP(&c.Name,
		"name", "n", "corechain", "block chain name")
	c.Cmd.Flags().StringVarP(&c.GenesisConf,
		"genesis_conf", "g", "./data/genesis/core.json", "genesis config file path")
	c.Cmd.Flags().StringVarP(&c.EnvConf,
		"env_conf", "e", "./conf/env.yaml", "env config file path")

	return c
}

func (c *CreateChainCommand) CreateChain() error {
	log.Printf("start create chain.bc_name:%s genesis_conf:%s env_conf:%s\n",
		c.Name, c.GenesisConf, c.EnvConf)

	if !utils.FileIsExist(c.GenesisConf) || !utils.FileIsExist(c.EnvConf) {
		log.Printf("config file not exist.genesis_conf:%s env_conf:%s\n", c.GenesisConf, c.EnvConf)
		return fmt.Errorf("config file not exist")
	}

	econf, err := xconf.LoadEnvConf(c.EnvConf)
	if err != nil {
		log.Printf("load env config failed.env_conf:%s err:%v\n", c.EnvConf, err)
		return fmt.Errorf("load env config failed")
	}

	logger.InitMLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))
	err = utils2.CreateLedger(c.Name, c.GenesisConf, econf)
	if err != nil {
		log.Printf("create ledger failed.err:%v\n", err)
		return fmt.Errorf("create ledger failed")
	}

	log.Printf("create ledger succ.bc_name:%s\n", c.Name)
	return nil
}
