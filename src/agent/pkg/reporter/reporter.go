// input: N/A (interface alias)
// output: reporter type alias to avoid circular dependency
// pos: reporter interface definition - if file updated, must sync with this header comment and pkg/reporter/CLAUDE.md
package reporter

import (
	"github.com/haolipeng/ebpf-based-microsegment/pkg/flow"
)

// Reporter is a type alias for the Reporter interface defined in the flow package
// This avoids circular dependencies while maintaining backward compatibility
type Reporter = flow.Reporter
