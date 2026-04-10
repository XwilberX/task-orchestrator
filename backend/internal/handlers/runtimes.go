package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
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
	key     string
	label   string
	image   string  // nombre en Docker Hub (puede ser distinto del namespace)
	hubRepo string  // repositorio en Docker Hub API
	suffix  string  // sufijo que se agrega al tag (ej: "-slim")
	pattern *regexp.Regexp
}

var runtimeDefs = []runtimeDef{
	{
		key:     "python",
		label:   "Python",
		image:   "python",
		hubRepo: "library/python",
		suffix:  "-slim",
		pattern: regexp.MustCompile(`^\d+\.\d+$`),
	},
	{
		key:     "nodejs",
		label:   "Node.js",
		image:   "node",
		hubRepo: "library/node",
		suffix:  "-slim",
		pattern: regexp.MustCompile(`^\d+$`),
	},
	{
		key:     "go",
		label:   "Go",
		image:   "golang",
		hubRepo: "library/golang",
		suffix:  "-alpine",
		pattern: regexp.MustCompile(`^\d+\.\d+$`),
	},
	{
		key:     "java",
		label:   "Java",
		image:   "eclipse-temurin",
		hubRepo: "library/eclipse-temurin",
		suffix:  "-alpine",
		pattern: regexp.MustCompile(`^\d+$`),
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

	// Fetch en paralelo
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

// fetchVersions consulta Docker Hub y devuelve versiones limpias (sin suffix).
func (h *RuntimesHandler) fetchVersions(def runtimeDef) ([]string, error) {
	url := fmt.Sprintf(
		"https://hub.docker.com/v2/repositories/%s/tags?page_size=100&ordering=-name",
		def.hubRepo,
	)

	resp, err := h.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	var versions []string

	for _, tag := range payload.Results {
		// El tag en Docker Hub incluye el suffix (ej: "3.11-slim")
		name := tag.Name
		if !strings.HasSuffix(name, def.suffix) {
			continue
		}
		// Quitar el suffix para obtener la versión limpia
		version := strings.TrimSuffix(name, def.suffix)
		if !def.pattern.MatchString(version) {
			continue
		}
		if _, dup := seen[version]; dup {
			continue
		}
		seen[version] = struct{}{}
		versions = append(versions, version)
	}

	// Ordenar descendente (semver simple: string sort funciona para major.minor)
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))

	// Limitar a 8 versiones máximo
	if len(versions) > 8 {
		versions = versions[:8]
	}

	return versions, nil
}
