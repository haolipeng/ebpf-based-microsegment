package reporter

import (
	"github.com/haolipeng/ebpf-based-microsegment/pkg/flow"
)

// Reporter is a type alias for the Reporter interface defined in the flow package
// This avoids circular dependencies while maintaining backward compatibility
type Reporter = flow.Reporter
