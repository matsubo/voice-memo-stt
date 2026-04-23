package engine

import "fmt"

var registry = map[string]Engine{}

func Register(e Engine) {
	registry[e.Name()] = e
}

func Get(name string) (Engine, error) {
	e, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown STT engine %q; registered: %v", name, registeredNames())
	}
	return e, nil
}

func registeredNames() []string {
	names := make([]string, 0, len(registry))
	for k := range registry {
		names = append(names, k)
	}
	return names
}
