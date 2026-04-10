package runtime

type Java struct{}

func (j *Java) Image(version string) string {
	if version == "" {
		version = "21"
	}
	return "eclipse-temurin:" + version + "-alpine"
}
func (j *Java) Entrypoint() string { return "Main.java" }

// Java no tiene gestor de paquetes sencillo — se ignora el campo packages.
func (j *Java) InstallCommand(_ string) string { return "" }

func (j *Java) RunCommand(entryPoint string, args []string) []string {
	dir := "/task"
	cmd := []string{"sh", "-c", "javac " + entryPoint + " && java -cp " + dir + " Main"}
	return append(cmd, args...)
}
