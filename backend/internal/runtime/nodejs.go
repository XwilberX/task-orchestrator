package runtime

import "strings"

type NodeJS struct{}

func (n *NodeJS) Image(version string) string {
	if version == "" {
		version = "20"
	}
	return "node:" + version + "-slim"
}
func (n *NodeJS) Entrypoint() string { return "main.js" }

func (n *NodeJS) InstallCommand(packages string) string {
	return "npm install --silent " + strings.TrimSpace(packages)
}

func (n *NodeJS) RunCommand(entryPoint string, args []string) []string {
	cmd := []string{"node", entryPoint}
	return append(cmd, args...)
}
