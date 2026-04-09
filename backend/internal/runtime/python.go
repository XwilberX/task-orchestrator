package runtime

import "strings"

type Python struct{}

func (p *Python) Image() string { return "python:3.11-slim" }
func (p *Python) Entrypoint() string { return "main.py" }

func (p *Python) InstallCommand(packages string) string {
	return "pip install --quiet " + strings.TrimSpace(packages)
}

func (p *Python) RunCommand(entryPoint string, args []string) []string {
	cmd := []string{"python", entryPoint}
	return append(cmd, args...)
}
