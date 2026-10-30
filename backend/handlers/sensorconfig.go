package handlers

import (
	"embed"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

var (
	errEmptyConfig = errors.New("sensor config template is empty")
	errNoProducers = errors.New("sensor config template has no producers.http section")
)

// sensordefaults holds Glutton's own config and rules; see its README for provenance.
//
//go:embed sensordefaults/config.yaml sensordefaults/rules.yaml
var sensordefaults embed.FS

// GetSensorConfig returns the Glutton config for one of the caller's sensors, with this
// instance's publish details filled in, so it can be dropped on the sensor host as-is.
//
// It is owner-only: the config carries the publish token, which is shared by the whole
// instance, so it must not be reachable by knowing a sensor id alone.
func (h *Handlers) GetSensorConfig(w http.ResponseWriter, r *http.Request) {
	if !h.sensorBelongsToCaller(w, r) {
		return
	}

	defaults, err := sensordefaults.ReadFile("sensordefaults/config.yaml")
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

// GetSensorRules returns the Glutton rules for one of the caller's sensors. They hold no
// secrets, but they sit behind the same check so both files are fetched the same way.
func (h *Handlers) GetSensorRules(w http.ResponseWriter, r *http.Request) {
	if !h.sensorBelongsToCaller(w, r) {
		return
	}

	rules, err := sensordefaults.ReadFile("sensordefaults/rules.yaml")
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	writeYAML(w, "rules.yaml", rules)
}

// sensorBelongsToCaller answers whether the sensor in the path exists and belongs to the
// caller, writing the response itself when it does not.
func (h *Handlers) sensorBelongsToCaller(w http.ResponseWriter, r *http.Request) bool {
	id := chi.URLParam(r, "id")
	if parsed, err := uuid.Parse(id); err != nil || parsed.String() != id {
		http.Error(w, "Invalid sensor id", http.StatusBadRequest)
		return false
	}

	if _, err := h.Sensors.GetSensor(id, userIDFromCtx(r.Context())); err != nil {
		if isNotFoundError(err) {
			// Someone else's sensor is reported the same way as one that does not exist,
			// so this does not say which ids are taken.
			http.Error(w, "Sensor not found", http.StatusNotFound)
			return false
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return false
	}
	return true
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
// owner made, so it is right behind a reverse proxy without configuring the public URL
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
