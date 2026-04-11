package executor

// RunConfig contiene toda la información necesaria para ejecutar una tarea.
type RunConfig struct {
	TaskID         string
	DefinitionName string
	Attempt        int
	Runtime        string
	RuntimeVersion string
	Code           string
	Args           []string
	Packages       string
	TimeoutSeconds int
	MemoryMB       int
	CPUShares      int
	NetworkEnabled bool
}

// ExecResult es el resultado de una ejecución de contenedor.
type ExecResult struct {
	ExitCode int
	Output   string // stdout + stderr combinados
	LastLine string // última línea no vacía de stdout (valor de retorno del script)
	TimedOut bool
}
