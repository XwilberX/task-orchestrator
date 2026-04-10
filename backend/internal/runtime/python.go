package runtime

import "strings"

type Python struct{}

func (p *Python) Image(version string) string {
	if version == "" {
		version = "3.11"
	}
	return "python:" + version + "-slim"
}
func (p *Python) Entrypoint() string { return "main.py" }

func (p *Python) InstallCommand(packages string) string {
	return "pip install --quiet " + strings.TrimSpace(packages)
}

func (p *Python) RunCommand(entryPoint string, args []string) []string {
	cmd := []string{"python", entryPoint}
	return append(cmd, args...)
}
