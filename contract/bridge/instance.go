package bridge

// InstanceCreatorConfig configures InstanceCreator
type InstanceCreatorConfig struct {
	Basedir        string
	SyscallService *SyscallService
	// VMConfig is the config of vm driver
	VMConfig VMConfig
}

// NewInstanceCreatorFunc instances a new InstanceCreator from InstanceCreatorConfig
type NewInstanceCreatorFunc func(config *InstanceCreatorConfig) (InstanceCreator, error)
