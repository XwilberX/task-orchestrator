package runtime

import "strings"

type Go struct{}

func (g *Go) Image(version string) string {
	if version == "" {
		version = "1.22"
	}
	return "golang:" + version + "-alpine"
}
func (g *Go) Entrypoint() string { return "main.go" }

func (g *Go) InstallCommand(packages string) string {
	return "go get " + strings.TrimSpace(packages)
}

func (g *Go) RunCommand(entryPoint string, args []string) []string {
	cmd := []string{"go", "run", entryPoint}
	return append(cmd, args...)
}
