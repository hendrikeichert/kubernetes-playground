package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type ServiceHealth struct {
	Service string         `json:"service"`
	Status  string         `json:"status"`
	Details map[string]any `json:"details,omitempty"`
}

type PodTarget struct {
	Name      string
	Namespace string
	Selector  string
}

type PodItem struct {
	Name            string `json:"name"`
	Phase           string `json:"phase"`
	ReadyContainers int    `json:"readyContainers"`
	TotalContainers int    `json:"totalContainers"`
	RestartCount    int    `json:"restartCount"`
}

type PodServiceStatus struct {
	Service   string    `json:"service"`
	Namespace string    `json:"namespace"`
	Selector  string    `json:"selector"`
	Status    string    `json:"status"`
	ReadyPods int       `json:"readyPods"`
	TotalPods int       `json:"totalPods"`
	Pods      []PodItem `json:"pods"`
	Error     string    `json:"error,omitempty"`
}

var (
	postgresHost    = getenv("POSTGRES_HOST", "postgresql.postgresql.svc.cluster.local")
	postgresPort    = getenvInt("POSTGRES_PORT", 5432)
	postgresTimeout = getenvDurationSeconds("POSTGRES_TIMEOUT", 2)

	mongodbHost    = getenv("MONGODB_HOST", "mongodb.mongodb.svc.cluster.local")
	mongodbPort    = getenvInt("MONGODB_PORT", 27017)
	mongodbTimeout = getenvDurationSeconds("MONGODB_TIMEOUT", 2)

	elasticsearchURL = getenv("ELASTICSEARCH_URL", "http://elasticsearch.elasticsearch.svc.cluster.local:9200/_cluster/health")
	httpTimeout      = getenvDurationSeconds("HTTP_TIMEOUT", 3)

	basePath = getenv("BASE_PATH", "/dashboard-api")

	kubeHost    = getenv("KUBERNETES_SERVICE_HOST", "kubernetes.default.svc")
	kubePort    = getenv("KUBERNETES_SERVICE_PORT", "443")
	kubeBaseURL = fmt.Sprintf("https://%s:%s", kubeHost, kubePort)

	saTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	saCAPath    = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

	podStatusTargets = []PodTarget{
		{Name: "PostgreSQL Service", Namespace: "postgresql", Selector: "app=postgresql"},
		{Name: "Adminer", Namespace: "webui", Selector: "app=adminer"},
		{Name: "MongoDB Service", Namespace: "mongodb", Selector: "app=mongodb"},
		{Name: "Elasticsearch Service", Namespace: "elasticsearch", Selector: "app=elasticsearch"},
		{Name: "Camunda Operate", Namespace: "camunda", Selector: "app.kubernetes.io/component=connectors"},
		{Name: "Camunda Tasklist", Namespace: "camunda", Selector: "app.kubernetes.io/component=connectors"},
		{Name: "Camunda Identity", Namespace: "camunda", Selector: "app.kubernetes.io/component=connectors"},
	}
)

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func mongodbHealth() ServiceHealth {
	address := fmt.Sprintf("%s:%d", mongodbHost, mongodbPort)
	connection, err := net.DialTimeout("tcp", address, mongodbTimeout)
	if err != nil {
		return ServiceHealth{
			Service: "mongodb",
			Status:  "DOWN",
			Details: map[string]any{"check": "tcp-connect", "host": mongodbHost, "port": mongodbPort, "error": err.Error()},
		}
	}
	_ = connection.Close()

	return ServiceHealth{
		Service: "mongodb",
		Status:  "UP",
		Details: map[string]any{"check": "tcp-connect", "host": mongodbHost, "port": mongodbPort},
	}
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvDurationSeconds(key string, fallback int) time.Duration {
	return time.Duration(getenvInt(key, fallback)) * time.Second
}

func postgresHealth() ServiceHealth {
	address := fmt.Sprintf("%s:%d", postgresHost, postgresPort)
	connection, err := net.DialTimeout("tcp", address, postgresTimeout)
	if err != nil {
		return ServiceHealth{
			Service: "postgresql",
			Status:  "DOWN",
			Details: map[string]any{"check": "tcp-connect", "host": postgresHost, "port": postgresPort, "error": err.Error()},
		}
	}
	_ = connection.Close()

	return ServiceHealth{
		Service: "postgresql",
		Status:  "UP",
		Details: map[string]any{"check": "tcp-connect", "host": postgresHost, "port": postgresPort},
	}
}

func elasticsearchHealth() ServiceHealth {
	client := &http.Client{Timeout: httpTimeout}
	request, err := http.NewRequest(http.MethodGet, elasticsearchURL, nil)
	if err != nil {
		return ServiceHealth{Service: "elasticsearch", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}

	response, err := client.Do(request)
	if err != nil {
		return ServiceHealth{Service: "elasticsearch", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	payload := map[string]any{}
	_ = json.Unmarshal(body, &payload)

	clusterStatus, _ := payload["status"].(string)
	status := "DOWN"
	if response.StatusCode == http.StatusOK {
		status = "UP"
	}

	return ServiceHealth{
		Service: "elasticsearch",
		Status:  status,
		Details: map[string]any{"httpStatus": response.StatusCode, "clusterStatus": clusterStatus},
	}
}

func readServiceAccountToken() (string, error) {
	data, err := os.ReadFile(saTokenPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func kubeGetJSON(path string, query url.Values) (map[string]any, error) {
	token, err := readServiceAccountToken()
	if err != nil {
		return nil, err
	}

	endpoint := kubeBaseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)

	caData, err := os.ReadFile(saCAPath)
	if err != nil {
		return nil, err
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("failed to parse kubernetes CA certificate")
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: roots},
		},
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("kubernetes api returned %d: %s", response.StatusCode, string(body))
	}

	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func summarizePod(raw map[string]any) PodItem {
	metadata, _ := raw["metadata"].(map[string]any)
	statusMap, _ := raw["status"].(map[string]any)

	containerStatuses, _ := statusMap["containerStatuses"].([]any)
	total := len(containerStatuses)
	ready := 0
	restarts := 0
	for _, item := range containerStatuses {
		container, _ := item.(map[string]any)
		readyValue, _ := container["ready"].(bool)
		if readyValue {
			ready++
		}
		restartValue, _ := container["restartCount"].(float64)
		restarts += int(restartValue)
	}

	name, _ := metadata["name"].(string)
	phase, _ := statusMap["phase"].(string)

	return PodItem{
		Name:            name,
		Phase:           phase,
		ReadyContainers: ready,
		TotalContainers: total,
		RestartCount:    restarts,
	}
}

func aggregatePodStatus(target PodTarget) PodServiceStatus {
	payload, err := kubeGetJSON(
		"/api/v1/namespaces/"+target.Namespace+"/pods",
		url.Values{"labelSelector": []string{target.Selector}},
	)
	if err != nil {
		return PodServiceStatus{
			Service:   target.Name,
			Namespace: target.Namespace,
			Selector:  target.Selector,
			Status:    "ERROR",
			ReadyPods: 0,
			TotalPods: 0,
			Pods:      []PodItem{},
			Error:     err.Error(),
		}
	}

	rawItems, _ := payload["items"].([]any)
	pods := make([]PodItem, 0, len(rawItems))
	readyPods := 0
	for _, item := range rawItems {
		podMap, _ := item.(map[string]any)
		pod := summarizePod(podMap)
		pods = append(pods, pod)
		if pod.Phase == "Running" && pod.ReadyContainers == pod.TotalContainers {
			readyPods++
		}
	}

	status := "DOWN"
	if len(pods) == 0 {
		status = "NO_PODS"
	} else if readyPods == len(pods) {
		status = "UP"
	} else if readyPods > 0 {
		status = "DEGRADED"
	}

	return PodServiceStatus{
		Service:   target.Name,
		Namespace: target.Namespace,
		Selector:  target.Selector,
		Status:    status,
		ReadyPods: readyPods,
		TotalPods: len(pods),
		Pods:      pods,
	}
}

func podStatusPayload() map[string]any {
	services := make([]PodServiceStatus, 0, len(podStatusTargets))
	overall := "UP"
	for _, target := range podStatusTargets {
		service := aggregatePodStatus(target)
		services = append(services, service)
		if service.Status != "UP" {
			overall = "DEGRADED"
		}
	}
	return map[string]any{"status": overall, "services": services}
}

func allHealthPayload() map[string]any {
	postgres := postgresHealth()
	mongodb := mongodbHealth()
	elasticsearch := elasticsearchHealth()
	overall := "UP"
	if postgres.Status != "UP" || mongodb.Status != "UP" || elasticsearch.Status != "UP" {
		overall = "DEGRADED"
	}
	return map[string]any{
		"status": overall,
		"services": map[string]any{
			"postgresql":    postgres,
			"mongodb":       mongodb,
			"elasticsearch": elasticsearch,
		},
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	data, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(statusCode)
	_, _ = w.Write(data)
}

func normalizePath(path string) string {
	clean := strings.SplitN(path, "?", 2)[0]
	if basePath != "" && strings.HasPrefix(clean, basePath) {
		stripped := strings.TrimPrefix(clean, basePath)
		if stripped == "" {
			return "/"
		}
		return stripped
	}
	return clean
}

func handler(w http.ResponseWriter, r *http.Request) {
	path := normalizePath(r.URL.String())

	switch path {
	case "/", "/health":
		writeJSON(w, http.StatusOK, allHealthPayload())
	case "/health/postgres":
		writeJSON(w, http.StatusOK, postgresHealth())
	case "/health/mongodb":
		writeJSON(w, http.StatusOK, mongodbHealth())
	case "/health/elasticsearch":
		writeJSON(w, http.StatusOK, elasticsearchHealth())
	case "/pod-status":
		writeJSON(w, http.StatusOK, podStatusPayload())
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not_found", "path": path})
	}
}

func main() {
	port := getenvInt("PORT", 8080)
	http.HandleFunc("/", handler)
	_ = http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil)
}
