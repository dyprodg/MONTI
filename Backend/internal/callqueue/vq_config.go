package callqueue

import "github.com/dennisdiepolder/monti/backend/internal/types"

// VQConfig holds the configuration for a virtual queue
type VQConfig struct {
	Name       types.VQName
	Department types.Department
	SLTarget   int // target percentage (e.g., 80)
	SLSeconds  int // threshold in seconds (e.g., 20)
}

// DefaultVQConfigs returns the default configuration for all 16 VQs
func DefaultVQConfigs() map[types.VQName]VQConfig {
	configs := make(map[types.VQName]VQConfig, 16)

	// All VQs default to 80/20 SL target
	for _, vq := range types.AllVQs {
		dept := types.VQDepartmentMapping[vq]
		configs[vq] = VQConfig{
			Name:       vq,
			Department: dept,
			SLTarget:   80,
			SLSeconds:  20,
		}
	}

	return configs
}
