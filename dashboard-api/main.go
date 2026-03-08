package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math"
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
	CPURequest      string `json:"cpuRequest,omitempty"`
	CPULimit        string `json:"cpuLimit,omitempty"`
	CPUUsage        string `json:"cpuUsage,omitempty"`
	MemoryRequest   string `json:"memoryRequest,omitempty"`
	MemoryLimit     string `json:"memoryLimit,omitempty"`
	MemoryUsage     string `json:"memoryUsage,omitempty"`
	CPURequestMilli int64  `json:"cpuRequestMilli,omitempty"`
	CPULimitMilli   int64  `json:"cpuLimitMilli,omitempty"`
	CPUUsageMilli   int64  `json:"cpuUsageMilli,omitempty"`
	MemoryRequestB  int64  `json:"memoryRequestBytes,omitempty"`
	MemoryLimitB    int64  `json:"memoryLimitBytes,omitempty"`
	MemoryUsageB    int64  `json:"memoryUsageBytes,omitempty"`
}

type PodServiceStatus struct {
	Service         string    `json:"service"`
	Namespace       string    `json:"namespace"`
	Selector        string    `json:"selector"`
	Status          string    `json:"status"`
	ReadyPods       int       `json:"readyPods"`
	TotalPods       int       `json:"totalPods"`
	CPURequest      string    `json:"cpuRequest,omitempty"`
	CPULimit        string    `json:"cpuLimit,omitempty"`
	CPUUsage        string    `json:"cpuUsage,omitempty"`
	MemoryRequest   string    `json:"memoryRequest,omitempty"`
	MemoryLimit     string    `json:"memoryLimit,omitempty"`
	MemoryUsage     string    `json:"memoryUsage,omitempty"`
	CPURequestMilli int64     `json:"cpuRequestMilli,omitempty"`
	CPULimitMilli   int64     `json:"cpuLimitMilli,omitempty"`
	CPUUsageMilli   int64     `json:"cpuUsageMilli,omitempty"`
	MemoryRequestB  int64     `json:"memoryRequestBytes,omitempty"`
	MemoryLimitB    int64     `json:"memoryLimitBytes,omitempty"`
	MemoryUsageB    int64     `json:"memoryUsageBytes,omitempty"`
	Pods            []PodItem `json:"pods"`
	Error           string    `json:"error,omitempty"`
}

var (
	postgresHost    = getenv("POSTGRES_HOST", "postgresql.postgresql.svc.cluster.local")
	postgresPort    = getenvInt("POSTGRES_PORT", 5432)
	postgresTimeout = getenvDurationSeconds("POSTGRES_TIMEOUT", 2)

	mongodbHost    = getenv("MONGODB_HOST", "mongodb.mongodb.svc.cluster.local")
	mongodbPort    = getenvInt("MONGODB_PORT", 27017)
	mongodbTimeout = getenvDurationSeconds("MONGODB_TIMEOUT", 2)

	elasticsearchURL       = getenv("ELASTICSEARCH_URL", "http://elasticsearch.elasticsearch.svc.cluster.local:9200/_cluster/health")
	otelCollectorHealthURL = getenv("OTEL_COLLECTOR_HEALTH_URL", "http://otel-collector.otel.svc.cluster.local:13133/")
	prometheusHealthURL    = getenv("PROMETHEUS_HEALTH_URL", "http://prometheus.prometheus.svc.cluster.local:9090/-/healthy")
	grafanaHealthURL       = getenv("GRAFANA_HEALTH_URL", "http://grafana.grafana.svc.cluster.local:3000/api/health")
	httpTimeout            = getenvDurationSeconds("HTTP_TIMEOUT", 3)

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
		{Name: "Confluent Kafka", Namespace: "confluent", Selector: "app=kafka"},
		{Name: "Confluent Schema Registry", Namespace: "confluent", Selector: "app=schema-registry"},
		{Name: "Redpanda Console", Namespace: "redpanda", Selector: "app=redpanda-console"},
		{Name: "Kafka Connect", Namespace: "kafka-connect", Selector: "app=kafka-connect"},
		{Name: "Camunda Connectors", Namespace: "camunda", Selector: "app.kubernetes.io/component=connectors"},
		{Name: "Camunda Zeebe", Namespace: "camunda", Selector: "app.kubernetes.io/component=zeebe-broker"},
		{Name: "OTel Collector", Namespace: "otel", Selector: "app=otel-collector"},
		{Name: "OTel Alloy", Namespace: "otel", Selector: "app=alloy"},
		{Name: "Prometheus", Namespace: "prometheus", Selector: "app=prometheus"},
		{Name: "Grafana", Namespace: "grafana", Selector: "app=grafana"},
		{Name: "Keycloak", Namespace: "keycloak", Selector: "app=keycloak"},
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
	address := net.JoinHostPort(mongodbHost, fmt.Sprintf("%d", mongodbPort))
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
	address := net.JoinHostPort(postgresHost, fmt.Sprintf("%d", postgresPort))
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

func otelCollectorHealth() ServiceHealth {
	client := &http.Client{Timeout: httpTimeout}
	request, err := http.NewRequest(http.MethodGet, otelCollectorHealthURL, nil)
	if err != nil {
		return ServiceHealth{Service: "otel-collector", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}

	response, err := client.Do(request)
	if err != nil {
		return ServiceHealth{Service: "otel-collector", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}
	defer response.Body.Close()

	status := "DOWN"
	if response.StatusCode == http.StatusOK {
		status = "UP"
	}

	return ServiceHealth{
		Service: "otel-collector",
		Status:  status,
		Details: map[string]any{"httpStatus": response.StatusCode, "url": otelCollectorHealthURL},
	}
}

func prometheusHealth() ServiceHealth {
	client := &http.Client{Timeout: httpTimeout}
	request, err := http.NewRequest(http.MethodGet, prometheusHealthURL, nil)
	if err != nil {
		return ServiceHealth{Service: "prometheus", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}

	response, err := client.Do(request)
	if err != nil {
		return ServiceHealth{Service: "prometheus", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}
	defer response.Body.Close()

	status := "DOWN"
	if response.StatusCode == http.StatusOK {
		status = "UP"
	}

	return ServiceHealth{
		Service: "prometheus",
		Status:  status,
		Details: map[string]any{"httpStatus": response.StatusCode, "url": prometheusHealthURL},
	}
}

func prometheusTargetsHealth() ServiceHealth {
	client := &http.Client{Timeout: httpTimeout}
	request, err := http.NewRequest(http.MethodGet, "http://prometheus.prometheus.svc.cluster.local:9090/api/v1/targets?state=active", nil)
	if err != nil {
		return ServiceHealth{Service: "prometheus-targets", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}

	response, err := client.Do(request)
	if err != nil {
		return ServiceHealth{Service: "prometheus-targets", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	payload := map[string]any{}
	_ = json.Unmarshal(body, &payload)

	data, _ := payload["data"].(map[string]any)
	activeTargets, _ := data["activeTargets"].([]any)

	upTargets := 0
	downTargets := 0
	for _, targetItem := range activeTargets {
		target, _ := targetItem.(map[string]any)
		health, _ := target["health"].(string)
		if strings.EqualFold(health, "up") {
			upTargets++
		} else {
			downTargets++
		}
	}

	status := "DOWN"
	if response.StatusCode == http.StatusOK && len(activeTargets) > 0 && downTargets == 0 {
		status = "UP"
	} else if response.StatusCode == http.StatusOK && len(activeTargets) > 0 && upTargets > 0 {
		status = "DEGRADED"
	}

	return ServiceHealth{
		Service: "prometheus-targets",
		Status:  status,
		Details: map[string]any{
			"httpStatus":    response.StatusCode,
			"activeTargets": len(activeTargets),
			"upTargets":     upTargets,
			"downTargets":   downTargets,
			"url":           "http://prometheus.prometheus.svc.cluster.local:9090/api/v1/targets?state=active",
		},
	}
}

func grafanaHealth() ServiceHealth {
	client := &http.Client{Timeout: httpTimeout}
	request, err := http.NewRequest(http.MethodGet, grafanaHealthURL, nil)
	if err != nil {
		return ServiceHealth{Service: "grafana", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}

	response, err := client.Do(request)
	if err != nil {
		return ServiceHealth{Service: "grafana", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}
	defer response.Body.Close()

	body, _ := io.ReadAll(response.Body)
	payload := map[string]any{}
	_ = json.Unmarshal(body, &payload)

	grafanaDBStatus, _ := payload["database"].(string)
	status := "DOWN"
	if response.StatusCode == http.StatusOK {
		status = "UP"
	}

	return ServiceHealth{
		Service: "grafana",
		Status:  status,
		Details: map[string]any{"httpStatus": response.StatusCode, "database": grafanaDBStatus, "url": grafanaHealthURL},
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

	cpuRequestMilli, cpuLimitMilli, memoryRequestBytes, memoryLimitBytes := parsePodResources(raw)

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
		CPURequest:      formatOptionalMillicores(cpuRequestMilli),
		CPULimit:        formatOptionalMillicores(cpuLimitMilli),
		CPUUsage:        "n/a",
		MemoryRequest:   formatOptionalBytesToMi(memoryRequestBytes),
		MemoryLimit:     formatOptionalBytesToMi(memoryLimitBytes),
		MemoryUsage:     "n/a",
		CPURequestMilli: cpuRequestMilli,
		CPULimitMilli:   cpuLimitMilli,
		MemoryRequestB:  memoryRequestBytes,
		MemoryLimitB:    memoryLimitBytes,
	}
}

func parsePodResources(raw map[string]any) (int64, int64, int64, int64) {
	spec, _ := raw["spec"].(map[string]any)
	containers, _ := spec["containers"].([]any)

	var cpuRequestMilli int64
	var cpuLimitMilli int64
	var memoryRequestBytes int64
	var memoryLimitBytes int64

	for _, container := range containers {
		containerMap, _ := container.(map[string]any)
		resources, _ := containerMap["resources"].(map[string]any)

		requests, _ := resources["requests"].(map[string]any)
		requestCPU, _ := requests["cpu"].(string)
		requestMemory, _ := requests["memory"].(string)
		cpuRequestMilli += parseCPUToMillicores(requestCPU)
		memoryRequestBytes += parseMemoryToBytes(requestMemory)

		limits, _ := resources["limits"].(map[string]any)
		limitCPU, _ := limits["cpu"].(string)
		limitMemory, _ := limits["memory"].(string)
		cpuLimitMilli += parseCPUToMillicores(limitCPU)
		memoryLimitBytes += parseMemoryToBytes(limitMemory)
	}

	return cpuRequestMilli, cpuLimitMilli, memoryRequestBytes, memoryLimitBytes
}

func parseCPUToMillicores(quantity string) int64 {
	if quantity == "" {
		return 0
	}
	if strings.HasSuffix(quantity, "m") {
		value, _ := strconv.ParseInt(strings.TrimSuffix(quantity, "m"), 10, 64)
		return value
	}
	if strings.HasSuffix(quantity, "n") {
		value, _ := strconv.ParseInt(strings.TrimSuffix(quantity, "n"), 10, 64)
		return value / 1000000
	}
	if strings.HasSuffix(quantity, "u") {
		value, _ := strconv.ParseInt(strings.TrimSuffix(quantity, "u"), 10, 64)
		return value / 1000
	}
	value, err := strconv.ParseFloat(quantity, 64)
	if err != nil {
		return 0
	}
	return int64(math.Round(value * 1000))
}

func parseMemoryToBytes(quantity string) int64 {
	if quantity == "" {
		return 0
	}

	binaryUnits := map[string]int64{
		"Ki": 1024,
		"Mi": 1024 * 1024,
		"Gi": 1024 * 1024 * 1024,
		"Ti": 1024 * 1024 * 1024 * 1024,
	}
	decimalUnits := map[string]int64{
		"K": 1000,
		"M": 1000 * 1000,
		"G": 1000 * 1000 * 1000,
		"T": 1000 * 1000 * 1000 * 1000,
	}

	for suffix, multiplier := range binaryUnits {
		if strings.HasSuffix(quantity, suffix) {
			value, _ := strconv.ParseFloat(strings.TrimSuffix(quantity, suffix), 64)
			return int64(math.Round(value * float64(multiplier)))
		}
	}

	for suffix, multiplier := range decimalUnits {
		if strings.HasSuffix(quantity, suffix) {
			value, _ := strconv.ParseFloat(strings.TrimSuffix(quantity, suffix), 64)
			return int64(math.Round(value * float64(multiplier)))
		}
	}

	value, err := strconv.ParseInt(quantity, 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func formatMillicores(millicores int64) string {
	if millicores <= 0 {
		return "0m"
	}
	return fmt.Sprintf("%dm", millicores)
}

func formatOptionalMillicores(millicores int64) string {
	if millicores <= 0 {
		return "n/a"
	}
	return fmt.Sprintf("%dm", millicores)
}

func formatBytesToMi(bytes int64) string {
	if bytes <= 0 {
		return "0Mi"
	}
	mi := float64(bytes) / (1024 * 1024)
	return fmt.Sprintf("%.0fMi", math.Round(mi))
}

func formatOptionalBytesToMi(bytes int64) string {
	if bytes <= 0 {
		return "n/a"
	}
	mi := float64(bytes) / (1024 * 1024)
	return fmt.Sprintf("%.0fMi", math.Round(mi))
}

func podUsageByName(namespace, selector string) (map[string]PodItem, int64, int64, error) {
	payload, err := kubeGetJSON(
		"/apis/metrics.k8s.io/v1beta1/namespaces/"+namespace+"/pods",
		url.Values{"labelSelector": []string{selector}},
	)
	if err != nil {
		return nil, 0, 0, err
	}

	items, _ := payload["items"].([]any)
	usageByPod := make(map[string]PodItem)
	var totalCPU int64
	var totalMemory int64

	for _, item := range items {
		podData, _ := item.(map[string]any)
		metadata, _ := podData["metadata"].(map[string]any)
		podName, _ := metadata["name"].(string)
		containers, _ := podData["containers"].([]any)

		var podCPU int64
		var podMemory int64
		for _, containerItem := range containers {
			containerMap, _ := containerItem.(map[string]any)
			usage, _ := containerMap["usage"].(map[string]any)
			cpuRaw, _ := usage["cpu"].(string)
			memoryRaw, _ := usage["memory"].(string)
			podCPU += parseCPUToMillicores(cpuRaw)
			podMemory += parseMemoryToBytes(memoryRaw)
		}

		totalCPU += podCPU
		totalMemory += podMemory
		usageByPod[podName] = PodItem{
			CPUUsage:      formatMillicores(podCPU),
			MemoryUsage:   formatBytesToMi(podMemory),
			CPUUsageMilli: podCPU,
			MemoryUsageB:  podMemory,
		}
	}

	return usageByPod, totalCPU, totalMemory, nil
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

	serviceCPURequestMilli := int64(0)
	serviceCPULimitMilli := int64(0)
	serviceMemoryRequestBytes := int64(0)
	serviceMemoryLimitBytes := int64(0)
	for _, pod := range pods {
		serviceCPURequestMilli += pod.CPURequestMilli
		serviceCPULimitMilli += pod.CPULimitMilli
		serviceMemoryRequestBytes += pod.MemoryRequestB
		serviceMemoryLimitBytes += pod.MemoryLimitB
	}

	serviceCPU := "n/a"
	serviceMemory := "n/a"
	serviceCPUUsageMilli := int64(0)
	serviceMemoryUsageBytes := int64(0)
	usageByPod, totalCPU, totalMemory, usageErr := podUsageByName(target.Namespace, target.Selector)
	if usageErr == nil {
		serviceCPU = formatMillicores(totalCPU)
		serviceMemory = formatBytesToMi(totalMemory)
		serviceCPUUsageMilli = totalCPU
		serviceMemoryUsageBytes = totalMemory
		for index := range pods {
			if usage, found := usageByPod[pods[index].Name]; found {
				pods[index].CPUUsage = usage.CPUUsage
				pods[index].MemoryUsage = usage.MemoryUsage
				pods[index].CPUUsageMilli = usage.CPUUsageMilli
				pods[index].MemoryUsageB = usage.MemoryUsageB
			}
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
		Service:         target.Name,
		Namespace:       target.Namespace,
		Selector:        target.Selector,
		Status:          status,
		ReadyPods:       readyPods,
		TotalPods:       len(pods),
		CPURequest:      formatOptionalMillicores(serviceCPURequestMilli),
		CPULimit:        formatOptionalMillicores(serviceCPULimitMilli),
		CPUUsage:        serviceCPU,
		MemoryRequest:   formatOptionalBytesToMi(serviceMemoryRequestBytes),
		MemoryLimit:     formatOptionalBytesToMi(serviceMemoryLimitBytes),
		MemoryUsage:     serviceMemory,
		CPURequestMilli: serviceCPURequestMilli,
		CPULimitMilli:   serviceCPULimitMilli,
		CPUUsageMilli:   serviceCPUUsageMilli,
		MemoryRequestB:  serviceMemoryRequestBytes,
		MemoryLimitB:    serviceMemoryLimitBytes,
		MemoryUsageB:    serviceMemoryUsageBytes,
		Pods:            pods,
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
	kafka := kafkaHealth()
	schemaRegistry := schemaRegistryHealth()
	redpandaConsole := redpandaConsoleHealth()
	kafkaConnect := kafkaConnectHealth()
	otelCollector := otelCollectorHealth()
	prometheus := prometheusHealth()
	grafana := grafanaHealth()
	overall := "UP"
	if postgres.Status != "UP" || mongodb.Status != "UP" || elasticsearch.Status != "UP" || kafka.Status != "UP" || schemaRegistry.Status != "UP" || redpandaConsole.Status != "UP" || kafkaConnect.Status != "UP" || otelCollector.Status != "UP" || prometheus.Status != "UP" || grafana.Status != "UP" {
		overall = "DEGRADED"
	}
	return map[string]any{
		"status": overall,
		"services": map[string]any{
			"postgresql":    postgres,
			"mongodb":       mongodb,
			"elasticsearch": elasticsearch,
			"kafka":         kafka,
			"schemaRegistry": schemaRegistry,
			"redpandaConsole": redpandaConsole,
			"kafkaConnect":  kafkaConnect,
			"otelCollector": otelCollector,
			"prometheus":    prometheus,
			"grafana":       grafana,
		},
	}
}

func swaggerSpecPayload() map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "Dashboard API",
			"version":     "1.0.0",
			"description": "Service overview and health endpoints for the cluster dashboard.",
		},
		"servers": []map[string]any{{"url": basePath}},
		"paths": map[string]any{
			"/health":                    map[string]any{"get": map[string]any{"summary": "Overall service health"}},
			"/health/postgres":           map[string]any{"get": map[string]any{"summary": "PostgreSQL health"}},
			"/health/mongodb":            map[string]any{"get": map[string]any{"summary": "MongoDB health"}},
			"/health/elasticsearch":      map[string]any{"get": map[string]any{"summary": "Elasticsearch health"}},
			"/health/kafka":              map[string]any{"get": map[string]any{"summary": "Confluent Kafka health"}},
			"/health/schema-registry":    map[string]any{"get": map[string]any{"summary": "Confluent Schema Registry health"}},
			"/health/redpanda-console":   map[string]any{"get": map[string]any{"summary": "Redpanda Console health"}},
			"/health/kafka-connect":      map[string]any{"get": map[string]any{"summary": "Kafka Connect health"}},
			"/health/otel":               map[string]any{"get": map[string]any{"summary": "OTel Collector health"}},
			"/health/prometheus":         map[string]any{"get": map[string]any{"summary": "Prometheus health"}},
			"/health/prometheus/targets": map[string]any{"get": map[string]any{"summary": "Prometheus scrape-target health"}},
			"/health/grafana":            map[string]any{"get": map[string]any{"summary": "Grafana health"}},
			"/pod-status":                map[string]any{"get": map[string]any{"summary": "Pod status summary by service"}},
			"/swagger":                   map[string]any{"get": map[string]any{"summary": "Swagger UI"}},
			"/swagger.json":              map[string]any{"get": map[string]any{"summary": "OpenAPI document"}},
		},
	}
}

func swaggerUIHTML() string {
	apiSpecURL := "/swagger.json"
	if basePath != "" {
		apiSpecURL = strings.TrimSuffix(basePath, "/") + "/swagger.json"
	}

	return fmt.Sprintf(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Dashboard API Swagger</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  </head>
  <body style="margin:0;">
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
      window.ui = SwaggerUIBundle({
        url: %q,
        dom_id: '#swagger-ui'
      });
    </script>
  </body>
</html>`, apiSpecURL)
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
	case "/health/kafka":
		writeJSON(w, http.StatusOK, kafkaHealth())
	case "/health/schema-registry":
		writeJSON(w, http.StatusOK, schemaRegistryHealth())
	case "/health/redpanda-console":
		writeJSON(w, http.StatusOK, redpandaConsoleHealth())
	case "/health/kafka-connect":
		writeJSON(w, http.StatusOK, kafkaConnectHealth())
	case "/health/otel":
		writeJSON(w, http.StatusOK, otelCollectorHealth())
	case "/health/prometheus":
		writeJSON(w, http.StatusOK, prometheusHealth())
	case "/health/prometheus/targets":
		writeJSON(w, http.StatusOK, prometheusTargetsHealth())
	case "/health/grafana":
		writeJSON(w, http.StatusOK, grafanaHealth())
	case "/health/dex":
		writeJSON(w, http.StatusOK, dexHealth())
	case "/pod-status":
		writeJSON(w, http.StatusOK, podStatusPayload())
	case "/swagger.json":
		writeJSON(w, http.StatusOK, swaggerSpecPayload())
	case "/swagger":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(swaggerUIHTML()))
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not_found", "path": path})
	}
}

func main() {
	port := getenvInt("PORT", 8080)
	http.HandleFunc("/", handler)
	_ = http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil)
}

func dexHealth() ServiceHealth {
	dexURL := "http://dex.dex.svc.cluster.local:5556/dex/healthz"
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(dexURL)
	if err != nil {
		return ServiceHealth{Service: "dex", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return ServiceHealth{Service: "dex", Status: "UP", Details: map[string]any{"httpStatus": resp.StatusCode, "url": dexURL}}
	}
	return ServiceHealth{Service: "dex", Status: "DOWN", Details: map[string]any{"httpStatus": resp.StatusCode, "url": dexURL}}
}

func kafkaHealth() ServiceHealth {
	kafkaHost := "kafka.confluent.svc.cluster.local"
	kafkaPort := 9092
	address := net.JoinHostPort(kafkaHost, fmt.Sprintf("%d", kafkaPort))
	connection, err := net.DialTimeout("tcp", address, httpTimeout)
	if err != nil {
		return ServiceHealth{
			Service: "kafka",
			Status:  "DOWN",
			Details: map[string]any{"check": "tcp-connect", "host": kafkaHost, "port": kafkaPort, "error": err.Error()},
		}
	}
	_ = connection.Close()

	return ServiceHealth{
		Service: "kafka",
		Status:  "UP",
		Details: map[string]any{"check": "tcp-connect", "host": kafkaHost, "port": kafkaPort},
	}
}

func schemaRegistryHealth() ServiceHealth {
	schemaRegistryURL := "http://schema-registry.confluent.svc.cluster.local:8081/subjects"
	client := &http.Client{Timeout: httpTimeout}
	request, err := http.NewRequest(http.MethodGet, schemaRegistryURL, nil)
	if err != nil {
		return ServiceHealth{Service: "schema-registry", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}

	response, err := client.Do(request)
	if err != nil {
		return ServiceHealth{Service: "schema-registry", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}
	defer response.Body.Close()

	status := "DOWN"
	if response.StatusCode == http.StatusOK {
		status = "UP"
	}

	return ServiceHealth{
		Service: "schema-registry",
		Status:  status,
		Details: map[string]any{"httpStatus": response.StatusCode, "url": schemaRegistryURL},
	}
}

func redpandaConsoleHealth() ServiceHealth {
	redpandaConsoleURL := "http://redpanda-console.redpanda.svc.cluster.local:8080"
	client := &http.Client{Timeout: httpTimeout}
	request, err := http.NewRequest(http.MethodGet, redpandaConsoleURL, nil)
	if err != nil {
		return ServiceHealth{Service: "redpanda-console", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}

	response, err := client.Do(request)
	if err != nil {
		return ServiceHealth{Service: "redpanda-console", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}
	defer response.Body.Close()

	status := "DOWN"
	if response.StatusCode >= 200 && response.StatusCode < 400 {
		status = "UP"
	}

	return ServiceHealth{
		Service: "redpanda-console",
		Status:  status,
		Details: map[string]any{"httpStatus": response.StatusCode, "url": redpandaConsoleURL},
	}
}

func kafkaConnectHealth() ServiceHealth {
	kafkaConnectURL := "http://kafka-connect.kafka-connect.svc.cluster.local:8083/connectors"
	client := &http.Client{Timeout: httpTimeout}
	request, err := http.NewRequest(http.MethodGet, kafkaConnectURL, nil)
	if err != nil {
		return ServiceHealth{Service: "kafka-connect", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}

	response, err := client.Do(request)
	if err != nil {
		return ServiceHealth{Service: "kafka-connect", Status: "DOWN", Details: map[string]any{"error": err.Error()}}
	}
	defer response.Body.Close()

	status := "DOWN"
	if response.StatusCode == http.StatusOK {
		status = "UP"
	}

	return ServiceHealth{
		Service: "kafka-connect",
		Status:  status,
		Details: map[string]any{"httpStatus": response.StatusCode, "url": kafkaConnectURL},
	}
}
