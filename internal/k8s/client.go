package k8s

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// Client wraps the K8s clientset for gateway status operations.
type Client struct {
	clientset kubernetes.Interface
	config    *rest.Config
	Name      string
}

// NewClient creates a K8s client from API server URL, bearer token, optional CA data, and insecure flag.
func NewClient(name, server, token, caData string, insecure bool) (*Client, error) {
	cfg := &rest.Config{
		Host:        server,
		BearerToken: token,
		Timeout:     30 * time.Second,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   decodeCAData(caData),
			Insecure: insecure,
		},
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("k8s: 创建 clientset 失败: %w", err)
	}

	return &Client{clientset: clientset, config: cfg, Name: name}, nil
}

func decodeCAData(caData string) []byte {
	if caData == "" {
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(caData)
	if err != nil {
		return nil
	}
	if !validCACert(decoded) {
		return nil
	}
	return decoded
}

func validCACert(pem []byte) bool {
	pool := x509.NewCertPool()
	return pool.AppendCertsFromPEM(pem)
}

// GetStatefulSetReplicas returns the replica count of a StatefulSet.
func (c *Client) GetStatefulSetReplicas(ctx context.Context, namespace, name string) (int, error) {
	sts, err := c.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("k8s: 获取 StatefulSet %s/%s 失败: %w", namespace, name, err)
	}
	if sts.Spec.Replicas == nil {
		return 0, nil
	}
	return int(*sts.Spec.Replicas), nil
}

// GetPodStatus returns the phase of a pod.
func (c *Client) GetPodStatus(ctx context.Context, namespace, podName string) (string, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("k8s: 获取 Pod %s/%s 失败: %w", namespace, podName, err)
	}
	return string(pod.Status.Phase), nil
}

// ExecCommand executes a command inside a container and returns stdout.
func (c *Client) ExecCommand(ctx context.Context, namespace, podName, container string, cmd []string) (string, error) {
	req := c.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	req.VersionedParams(&corev1.PodExecOptions{
		Container: container,
		Command:   cmd,
		Stdout:    true,
		Stderr:    false,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(c.config, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("k8s: 创建 SPDY executor 失败: %w", err)
	}

	var stdout strings.Builder
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: nil,
	})
	if err != nil {
		return "", fmt.Errorf("k8s: exec 命令失败: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}
