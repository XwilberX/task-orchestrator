package runtime

import (
	"fmt"
	"strings"
)

type Java struct{}

func (j *Java) Image(version string) string {
	if version == "" {
		version = "21"
	}
	return "eclipse-temurin:" + version
}
func (j *Java) Entrypoint() string { return "Main.java" }

// InstallCommand descarga JARs desde Maven Central.
// El campo packages acepta coordenadas Maven separadas por espacios:
//
//	com.google.code.gson:gson:2.10.1 org.apache.commons:commons-lang3:3.14.0
func (j *Java) InstallCommand(packages string) string {
	if strings.TrimSpace(packages) == "" {
		return ""
	}
	cmds := []string{"mkdir -p /task/libs"}
	for _, pkg := range strings.Fields(packages) {
		parts := strings.Split(pkg, ":")
		if len(parts) != 3 {
			continue
		}
		groupPath := strings.ReplaceAll(parts[0], ".", "/")
		artifactID := parts[1]
		version := parts[2]
		url := fmt.Sprintf(
			"https://repo1.maven.org/maven2/%s/%s/%s/%s-%s.jar",
			groupPath, artifactID, version, artifactID, version,
		)
		cmds = append(cmds, fmt.Sprintf(
			"curl -sSL -o /task/libs/%s-%s.jar %s", artifactID, version, url,
		))
	}
	return strings.Join(cmds, " && ")
}

func (j *Java) RunCommand(entryPoint string, _ []string) []string {
	return []string{"sh", "-c",
		"mkdir -p /task/libs && " +
			"javac -cp '/task/libs/*' " + entryPoint + " && " +
			"java -cp '/task:/task/libs/*' Main",
	}
}
