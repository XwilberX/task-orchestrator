package runtime

import "fmt"

// Runtime define la interfaz que cada lenguaje debe implementar.
type Runtime interface {
	Image(version string) string
	InstallCommand(packages string) string
	RunCommand(entryPoint string, args []string) []string
	Entrypoint() string // nombre del archivo a inyectar (ej: main.py)
}

// Get devuelve el Runtime para el nombre dado o error si no existe.
func Get(name string) (Runtime, error) {
	switch name {
	case "python":
		return &Python{}, nil
	case "nodejs":
		return &NodeJS{}, nil
	case "go":
		return &Go{}, nil
	case "java":
		return &Java{}, nil
	default:
		return nil, fmt.Errorf("runtime desconocido: %s", name)
	}
}
