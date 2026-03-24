package runtime

import (
	"time"

	"github.com/dunialabs/kimbap-core/internal/adapters"
)

func NewRuntime(rt Runtime) *Runtime {
	if rt.Adapters == nil {
		rt.Adapters = map[string]adapters.Adapter{}
	}
	if rt.Now == nil {
		rt.Now = time.Now
	}
	return &rt
}
