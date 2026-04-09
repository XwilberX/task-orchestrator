package runtime

import "strings"

type TypeScript struct{}

func (t *TypeScript) Image() string { return "node:20-slim" }
func (t *TypeScript) Entrypoint() string { return "main.ts" }

func (t *TypeScript) InstallCommand(packages string) string {
	base := "npm install -g tsx --silent"
	if pkg := strings.TrimSpace(packages); pkg != "" {
		base += " && npm install --silent " + pkg
	}
	return base
}

func (t *TypeScript) RunCommand(entryPoint string, args []string) []string {
	cmd := []string{"tsx", entryPoint}
	return append(cmd, args...)
}
