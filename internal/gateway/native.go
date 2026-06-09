package gateway

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"bbc-mcp/internal/config"
	"bbc-mcp/internal/k8s"
)

// PodConnectionInfo holds the connection count for a single gateway pod.
type PodConnectionInfo struct {
	PodName     string `json:"pod_name"`
	Status      string `json:"status"`
	Connections int    `json:"connections"`
	Devices     int    `json:"devices"`
	Error       string `json:"error,omitempty"`
}

// ClusterGatewayStatus holds the gateway status for a single K8s cluster.
type ClusterGatewayStatus struct {
	Cluster string              `json:"cluster"`
	Pods    []PodConnectionInfo `json:"pods"`
}

// QueryGatewayStatus queries gateway connection counts on a single K8s cluster.
func QueryGatewayStatus(ctx context.Context, client *k8s.Client, cfg config.GatewayNativeConfig) (*ClusterGatewayStatus, error) {
	replicas, err := client.GetStatefulSetReplicas(ctx, cfg.Namespace, cfg.StatefulSet)
	if err != nil {
		return nil, err
	}

	result := &ClusterGatewayStatus{
		Cluster: client.Name,
		Pods:    make([]PodConnectionInfo, replicas),
	}

	var wg sync.WaitGroup
	for i := 0; i < replicas; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			podName := fmt.Sprintf("%s-%d", cfg.StatefulSet, idx)
			info := queryPodConnections(ctx, client, cfg.Namespace, podName, cfg.Container, cfg.Port)
			result.Pods[idx] = info
		}(i)
	}
	wg.Wait()

	return result, nil
}

func queryPodConnections(ctx context.Context, client *k8s.Client, namespace, podName, container string, port int) PodConnectionInfo {
	info := PodConnectionInfo{PodName: podName}

	status, err := client.GetPodStatus(ctx, namespace, podName)
	if err != nil {
		info.Status = "Error"
		info.Error = err.Error()
		return info
	}
	info.Status = status

	if status != "Running" {
		info.Error = fmt.Sprintf("Pod 状态为 %s，非 Running", status)
		return info
	}

	cmd := []string{
		"sh", "-c",
		fmt.Sprintf("netstat -antp 2>/dev/null | grep EST | grep ':%d' | wc -l", port),
	}
	output, err := client.ExecCommand(ctx, namespace, podName, container, cmd)
	if err != nil {
		info.Status = "Error"
		info.Error = err.Error()
		return info
	}

	connections, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		info.Error = fmt.Sprintf("解析连接数失败: %s", output)
		return info
	}

	info.Connections = connections
	info.Devices = connections / 3
	return info
}
