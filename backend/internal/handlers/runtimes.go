package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/XwilberX/task-orchestrator/pkg/response"
)

// runtimeMeta describe un runtime disponible.
type runtimeMeta struct {
	Label    string   `json:"label"`
	Image    string   `json:"image"`
	Suffix   string   `json:"suffix"`
	Versions []string `json:"versions"`
}

// runtimeDef es la configuración estática de cada runtime para consultar Docker Hub.
type runtimeDef struct {
	key         string
	label       string
	image       string
	hubRepo     string
	suffix      string         // sufijo en el tag de Docker Hub (ej: "-slim")
	pattern     *regexp.Regexp // patrón para validar la versión (sin suffix)
	searchTerms []string       // términos de búsqueda para filtrar tags en Docker Hub
}

var runtimeDefs = []runtimeDef{
	{
		key:     "python",
		label:   "Python",
		image:   "python",
		hubRepo: "library/python",
		suffix:  "-slim",
		pattern: regexp.MustCompile(`^\d+\.\d+$`),
		// Buscamos tags que contengan "3." para obtener versiones Python 3.x
		searchTerms: []string{"3."},
	},
	{
		key:     "nodejs",
		label:   "Node.js",
		image:   "node",
		hubRepo: "library/node",
		suffix:  "-slim",
		pattern: regexp.MustCompile(`^\d+$`),
		// Buscamos por cada major version de Node LTS y actuales
		searchTerms: []string{"18-", "20-", "22-", "23-", "24-"},
	},
	{
		key:     "go",
		label:   "Go",
		image:   "golang",
		hubRepo: "library/golang",
		suffix:  "-alpine",
		pattern: regexp.MustCompile(`^\d+\.\d+$`),
		// Buscamos por minor versions recientes de Go 1.x
		searchTerms: []string{"1.24", "1.23", "1.22", "1.21", "1.20", "1.19"},
	},
	{
		key:     "java",
		label:   "Java",
		image:   "eclipse-temurin",
		hubRepo: "library/eclipse-temurin",
		suffix:  "",
		pattern: regexp.MustCompile(`^\d+$`),
		// eclipse-temurin no tiene tags -alpine, usamos tag plain (21, 17, 11, 8)
		searchTerms: []string{"8", "11", "17", "21", "23"},
	},
}

// versionsCache cachea los resultados de Docker Hub durante 1 hora.
type versionsCache struct {
	mu        sync.RWMutex
	data      map[string]runtimeMeta
	fetchedAt time.Time
}

var cache = &versionsCache{}

const cacheTTL = time.Hour

// RuntimesHandler sirve GET /api/v1/runtimes
type RuntimesHandler struct {
	client *http.Client
}

func NewRuntimesHandler() *RuntimesHandler {
	return &RuntimesHandler{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (h *RuntimesHandler) List(w http.ResponseWriter, r *http.Request) {
	data, err := h.getRuntimes(r.Context())
	if err != nil {
		response.InternalError(w, err)
		return
	}
	response.OK(w, data, "")
}

func (h *RuntimesHandler) getRuntimes(ctx interface{ Done() <-chan struct{} }) (map[string]runtimeMeta, error) {
	cache.mu.RLock()
	if cache.data != nil && time.Since(cache.fetchedAt) < cacheTTL {
		data := cache.data
		cache.mu.RUnlock()
		return data, nil
	}
	cache.mu.RUnlock()

	type result struct {
		key  string
		meta runtimeMeta
		err  error
	}

	ch := make(chan result, len(runtimeDefs))
	for _, def := range runtimeDefs {
		def := def
		go func() {
			versions, err := h.fetchVersions(def)
			ch <- result{
				key: def.key,
				meta: runtimeMeta{
					Label:    def.label,
					Image:    def.image,
					Suffix:   def.suffix,
					Versions: versions,
				},
				err: err,
			}
		}()
	}

	data := make(map[string]runtimeMeta)
	var firstErr error
	for range runtimeDefs {
		r := <-ch
		if r.err != nil {
			firstErr = r.err
			continue
		}
		if len(r.meta.Versions) > 0 {
			data[r.key] = r.meta
		}
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no se pudieron obtener versiones de Docker Hub: %v", firstErr)
	}

	cache.mu.Lock()
	cache.data = data
	cache.fetchedAt = time.Now()
	cache.mu.Unlock()

	return data, nil
}

// fetchVersions consulta Docker Hub usando múltiples términos de búsqueda por major/minor version.
func (h *RuntimesHandler) fetchVersions(def runtimeDef) ([]string, error) {
	seen := map[string]struct{}{}
	var versions []string

	for _, term := range def.searchTerms {
		url := fmt.Sprintf(
			"https://hub.docker.com/v2/repositories/%s/tags?page_size=50&name=%s",
			def.hubRepo, term,
		)

		resp, err := h.client.Get(url)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		var payload struct {
			Results []struct {
				Name string `json:"name"`
			} `json:"results"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			continue
		}

		for _, tag := range payload.Results {
			name := tag.Name
			// Si hay suffix, el tag debe terminar exactamente con él
			var version string
			if def.suffix != "" {
				if len(name) <= len(def.suffix) {
					continue
				}
				if name[len(name)-len(def.suffix):] != def.suffix {
					continue
				}
				version = name[:len(name)-len(def.suffix)]
			} else {
				version = name
			}

			if !def.pattern.MatchString(version) {
				continue
			}
			if _, dup := seen[version]; dup {
				continue
			}
			seen[version] = struct{}{}
			versions = append(versions, version)
		}
	}

	// Ordenar descendente por versión semántica (numérico, no lexicográfico)
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j]) > 0
	})

	if len(versions) > 10 {
		versions = versions[:10]
	}

	return versions, nil
}

// compareVersions compara dos versiones numéricas (ej: "3.12" vs "3.9", "21" vs "8").
// Devuelve >0 si a > b, <0 si a < b, 0 si son iguales.
func compareVersions(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")
	max := len(partsA)
	if len(partsB) > max {
		max = len(partsB)
	}
	for i := 0; i < max; i++ {
		var na, nb int
		if i < len(partsA) {
			na, _ = strconv.Atoi(partsA[i])
		}
		if i < len(partsB) {
			nb, _ = strconv.Atoi(partsB[i])
		}
		if na != nb {
			return na - nb
		}
	}
	return 0
}
