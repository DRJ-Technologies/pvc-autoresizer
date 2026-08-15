package runners

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/topolvm/pvc-autoresizer/internal/metrics"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// NewK8sMetricsApiClient returns a new k8sMetricsApiClient client
func NewK8sMetricsApiClient() (MetricsClient, error) {
	return &k8sMetricsApiClient{}, nil
}

type k8sMetricsApiClient struct {
}

func nodeWasDeleted(ctx context.Context, clientset kubernetes.Interface, nodeName string) (bool, error) {
	_, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, v1.GetOptions{})
	if err == nil {
		return false, nil
	}
	if apierrors.IsNotFound(err) {
		return true, nil
	}
	return false, err
}

type nodeMetricsFetcher func(context.Context, string) (map[types.NamespacedName]*VolumeStats, error)

func collectPVCUsage(
	ctx context.Context,
	clientset kubernetes.Interface,
	nodes []corev1.Node,
	fetch nodeMetricsFetcher,
) (map[types.NamespacedName]*VolumeStats, error) {
	pvcUsage := make(map[types.NamespacedName]*VolumeStats)
	var mu sync.Mutex

	eg, ctx := errgroup.WithContext(ctx)
	for _, node := range nodes {
		nodeName := node.Name
		eg.Go(func() error {
			nodePVCUsage, err := fetch(ctx, nodeName)
			if err != nil {
				deleted, verificationErr := nodeWasDeleted(ctx, clientset, nodeName)
				if verificationErr != nil {
					return fmt.Errorf(
						"failed to verify node %s after kubelet metrics error %v: %w",
						nodeName, err, verificationErr,
					)
				}
				if deleted {
					metrics.MetricsClientDeletedNodeSkippedTotal.Increment()
					return nil
				}
				return err
			}

			mu.Lock()
			defer mu.Unlock()
			for k, v := range nodePVCUsage {
				pvcUsage[k] = v
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return pvcUsage, nil
}

func (c *k8sMetricsApiClient) GetMetrics(ctx context.Context) (map[types.NamespacedName]*VolumeStats, error) {
	// create a Kubernetes client using in-cluster configuration
	config, err := rest.InClusterConfig()
	if err != nil {
		metrics.MetricsClientFailTotal.Increment()
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		metrics.MetricsClientFailTotal.Increment()
		return nil, err
	}

	// get a list of nodes and IP addresses
	nodes, err := clientset.CoreV1().Nodes().List(ctx, v1.ListOptions{})
	if err != nil {
		metrics.MetricsClientFailTotal.Increment()
		return nil, err
	}

	pvcUsage, err := collectPVCUsage(
		ctx,
		clientset,
		nodes.Items,
		func(ctx context.Context, nodeName string) (map[types.NamespacedName]*VolumeStats, error) {
			return getPVCUsageFromK8sMetricsAPI(ctx, clientset, nodeName)
		},
	)
	if err != nil {
		metrics.MetricsClientFailTotal.Increment()
		return nil, err
	}

	return pvcUsage, nil
}

func getPVCUsageFromK8sMetricsAPI(
	ctx context.Context, clientset *kubernetes.Clientset, nodeName string,
) (map[types.NamespacedName]*VolumeStats, error) {
	// make the request to the api /metrics endpoint and handle the response
	req := clientset.
		CoreV1().
		RESTClient().
		Get().
		Resource("nodes").
		Name(nodeName).
		SubResource("proxy").
		Suffix("metrics")
	respBody, err := req.DoRaw(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats from kubelet on node %s: %w", nodeName, err)
	}
	parser := expfmt.NewTextParser(model.UTF8Validation)
	metricFamilies, err := parser.TextToMetricFamilies(bytes.NewReader(respBody))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from kubelet on node %s: %w", nodeName, err)
	}

	pvcUsage := make(map[types.NamespacedName]*VolumeStats)

	// volumeAvailableQuery
	if gauge, ok := metricFamilies[volumeAvailableQuery]; ok {
		for _, m := range gauge.Metric {
			pvcName, value := parseMetric(m)
			pvcUsage[pvcName] = &VolumeStats{}
			pvcUsage[pvcName].AvailableBytes = int64(value)
		}
	}
	// volumeCapacityQuery
	if gauge, ok := metricFamilies[volumeCapacityQuery]; ok {
		for _, m := range gauge.Metric {
			pvcName, value := parseMetric(m)
			pvcUsage[pvcName].CapacityBytes = int64(value)
		}
	}

	// inodesAvailableQuery
	if gauge, ok := metricFamilies[inodesAvailableQuery]; ok {
		for _, m := range gauge.Metric {
			pvcName, value := parseMetric(m)
			pvcUsage[pvcName].AvailableInodeSize = int64(value)
		}
	}

	// inodesCapacityQuery
	if gauge, ok := metricFamilies[inodesCapacityQuery]; ok {
		for _, m := range gauge.Metric {
			pvcName, value := parseMetric(m)
			pvcUsage[pvcName].CapacityInodeSize = int64(value)
		}
	}
	return pvcUsage, nil
}

func parseMetric(m *dto.Metric) (pvcName types.NamespacedName, value uint64) {
	for _, label := range m.GetLabel() {
		if label.GetName() == "namespace" {
			pvcName.Namespace = label.GetValue()
		} else if label.GetName() == "persistentvolumeclaim" {
			pvcName.Name = label.GetValue()
		}
	}
	value = uint64(m.GetGauge().GetValue())
	return pvcName, value
}
