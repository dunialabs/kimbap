package registry

import "fmt"

type ErrNotFound struct {
	Name     string
	Registry string
}

func (e *ErrNotFound) Error() string {
	if e.Registry != "" {
		return fmt.Sprintf("service %q not found in registry %q", e.Name, e.Registry)
	}
	return fmt.Sprintf("service %q not found", e.Name)
}
