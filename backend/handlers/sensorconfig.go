package handlers

import (
	"embed"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

var (
	errEmptyConfig = errors.New("sensor config template is empty")
	errNoProducers = errors.New("sensor config template has no producers.http section")
)

// gluttonRef pins the Glutton revision the default config and rules are taken from. The
// files are fetched from the Glutton repository at this revision rather than vendored, so
// they are not a second copy that drifts; the embedded files below are only a fallback
// for when the fetch cannot be made.
const gluttonRef = "54fd4c9f578bbb9f1d77093b4ddeb04b28771422"

const gluttonRawBase = "https://raw.githubusercontent.com/mushorg/glutton/" + gluttonRef + "/config/"

// sensordefaults is the offline fallback for the files fetched from Glutton; see its
// README for provenance. It is served only when the fetch fails, so the endpoints keep
// working with no network.
//
//go:embed sensordefaults/config.yaml sensordefaults/rules.yaml
var sensordefaults embed.FS

// GetSensorConfig returns the Glutton config for the sensor named in the path, with this
// instance's publish details filled in, so it can be dropped on the sensor host as-is.
//
// The caller authenticates as the sensor, with the sensor's token (see sensorFromToken).
// The config carries the publish token, so it must only reach the sensor it belongs to.
func (h *Handlers) GetSensorConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.sensorFromToken(w, r); !ok {
		return
	}

	defaults, err := h.gluttonDefault(r, "config.yaml")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	config, err := withPublisher(defaults, publishURL(r, h.PublishToken))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	writeYAML(w, "config.yaml", config)
}

// GetSensorRules returns the Glutton rules for the sensor named in the path. They hold no
// secrets, but they go through the same token check so both files are fetched the same way.
func (h *Handlers) GetSensorRules(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.sensorFromToken(w, r); !ok {
		return
	}

	rules, err := h.gluttonDefault(r, "rules.yaml")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	writeYAML(w, "rules.yaml", rules)
}

// sensorFromToken authenticates a request made by a sensor for its own files and returns
// that sensor. The sensor sends its token in an Authorization: Bearer header; the token
// must belong to a sensor, and that sensor must be the one named in the path. A missing
// or unknown token is a 401, and a token for a different sensor is reported as not found,
// so a valid token for one sensor cannot read another's config and the response does not
// reveal which ids exist.
func (h *Handlers) sensorFromToken(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := chi.URLParam(r, "id")
	if parsed, err := uuid.Parse(id); err != nil || parsed.String() != id {
		http.Error(w, "Invalid sensor id", http.StatusBadRequest)
		return "", false
	}

	token := sensorBearerToken(r)
	if token == "" {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return "", false
	}

	sensor, err := h.Sensors.GetSensorByToken(token)
	if err != nil {
		if isNotFoundError(err) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return "", false
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return "", false
	}
	if sensor.ID != id {
		http.Error(w, "Sensor not found", http.StatusNotFound)
		return "", false
	}
	return sensor.ID, true
}

// sensorBearerToken returns the value of an Authorization: Bearer header, matched
// case-insensitively per RFC 7235, or an empty string.
func sensorBearerToken(r *http.Request) string {
	const prefix = "bearer "
	auth := r.Header.Get("Authorization")
	if len(auth) > len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
		return auth[len(prefix):]
	}
	return ""
}

// gluttonDefault returns one of Glutton's default files, fetched from the pinned revision
// and cached, or the embedded copy when the fetch fails so the endpoint works offline.
func (h *Handlers) gluttonDefault(r *http.Request, name string) ([]byte, error) {
	if cached, ok := h.defaultsCache.Load(name); ok {
		return cached.([]byte), nil
	}
	if body, err := fetchGluttonDefault(r, h.HTTPClient, name); err == nil {
		h.defaultsCache.Store(name, body)
		return body, nil
	}
	return sensordefaults.ReadFile("sensordefaults/" + name)
}

// fetchGluttonDefault downloads one file from the Glutton repository at the pinned ref.
func fetchGluttonDefault(r *http.Request, client *http.Client, name string) ([]byte, error) {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, gluttonRawBase+name, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("fetch glutton default: " + resp.Status)
	}
	// The files are a few KiB; cap the read so a wrong URL cannot stream without bound.
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

func writeYAML(w http.ResponseWriter, filename string, body []byte) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	// Browsers save it under the name Glutton expects instead of rendering it.
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	if _, err := w.Write(body); err != nil {
		return
	}
}

// publishURL builds the address this instance receives events on, from the request the
// sensor made, so it is right behind a reverse proxy without configuring the public URL
// twice. A forged Host only affects the config returned to the caller who forged it.
func publishURL(r *http.Request, token string) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	host := r.Host
	if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" {
		host = forwarded
	}
	u := url.URL{Scheme: scheme, Host: host, Path: "/api/publish"}
	if token != "" {
		u.RawQuery = url.Values{"token": {token}}.Encode()
	}
	return u.String()
}

// withPublisher turns on the HTTP producer and points it at remote. It edits the parsed
// document rather than the struct so Glutton's own comments and key order survive.
func withPublisher(config []byte, remote string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(config, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 {
		return nil, errEmptyConfig
	}

	producers := mappingValue(doc.Content[0], "producers")
	if producers == nil {
		return nil, errNoProducers
	}
	httpProducer := mappingValue(producers, "http")
	if httpProducer == nil {
		return nil, errNoProducers
	}

	setScalar(producers, "enabled", "true", "!!bool")
	setScalar(httpProducer, "enabled", "true", "!!bool")
	setScalar(httpProducer, "remote", remote, "!!str")

	var out strings.Builder
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return []byte(out.String()), nil
}

// mappingValue returns the value node for key in a mapping, or nil.
func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	// A mapping's children alternate key, value.
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// setScalar sets key to value in a mapping, adding it when it is missing.
func setScalar(node *yaml.Node, key, value, tag string) {
	if existing := mappingValue(node, key); existing != nil {
		existing.Kind = yaml.ScalarNode
		existing.Tag = tag
		existing.Value = value
		existing.Content = nil
		return
	}
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value},
	)
}
