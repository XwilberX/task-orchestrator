package runtime

type Java struct{}

func (j *Java) Image() string { return "eclipse-temurin:21-alpine" }
func (j *Java) Entrypoint() string { return "Main.java" }

// Java no tiene gestor de paquetes sencillo — se ignora el campo packages.
func (j *Java) InstallCommand(_ string) string { return "" }

func (j *Java) RunCommand(entryPoint string, args []string) []string {
	// javac Main.java && java -cp /task Main
	dir := "/task"
	cmd := []string{"sh", "-c", "javac " + entryPoint + " && java -cp " + dir + " Main"}
	return append(cmd, args...)
}
