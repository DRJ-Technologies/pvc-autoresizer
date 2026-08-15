package runners

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestNodeWasDeleted(t *testing.T) {
	t.Run("deleted node", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		deleted, err := nodeWasDeleted(context.Background(), clientset, "deleted-node")
		if err != nil {
			t.Fatalf("nodeWasDeleted returned an unexpected error: %v", err)
		}
		if !deleted {
			t.Fatal("nodeWasDeleted returned false for a missing node")
		}
	})

	t.Run("live node", func(t *testing.T) {
		clientset := fake.NewSimpleClientset(&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "live-node"},
		})
		deleted, err := nodeWasDeleted(context.Background(), clientset, "live-node")
		if err != nil {
			t.Fatalf("nodeWasDeleted returned an unexpected error: %v", err)
		}
		if deleted {
			t.Fatal("nodeWasDeleted returned true for a live node")
		}
	})

	t.Run("unverifiable node", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		verificationErr := errors.New("node API unavailable")
		clientset.PrependReactor("get", "nodes", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, verificationErr
		})

		deleted, err := nodeWasDeleted(context.Background(), clientset, "unknown-node")
		if deleted {
			t.Fatal("nodeWasDeleted returned true when node state could not be verified")
		}
		if !errors.Is(err, verificationErr) {
			t.Fatalf("nodeWasDeleted returned %v, want %v", err, verificationErr)
		}
	})
}

func TestCollectPVCUsage(t *testing.T) {
	metricsErr := errors.New("kubelet metrics unavailable")

	t.Run("skips a deleted node and keeps metrics from live nodes", func(t *testing.T) {
		liveNode := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "live-node"}}
		clientset := fake.NewSimpleClientset(liveNode)
		pvcName := types.NamespacedName{Namespace: "valkey", Name: "data"}
		nodes := []corev1.Node{
			{ObjectMeta: metav1.ObjectMeta{Name: "deleted-node"}},
			*liveNode,
		}

		usage, err := collectPVCUsage(
			context.Background(),
			clientset,
			nodes,
			func(_ context.Context, nodeName string) (map[types.NamespacedName]*VolumeStats, error) {
				if nodeName == "deleted-node" {
					return nil, metricsErr
				}
				return map[types.NamespacedName]*VolumeStats{
					pvcName: {CapacityBytes: 100, AvailableBytes: 40},
				}, nil
			},
		)
		if err != nil {
			t.Fatalf("collectPVCUsage returned an unexpected error: %v", err)
		}
		if got := usage[pvcName]; got == nil || got.CapacityBytes != 100 || got.AvailableBytes != 40 {
			t.Fatalf("collectPVCUsage returned unexpected live-node metrics: %#v", got)
		}
	})

	t.Run("fails closed when the node still exists", func(t *testing.T) {
		liveNode := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "live-node"}}
		clientset := fake.NewSimpleClientset(liveNode)
		_, err := collectPVCUsage(
			context.Background(),
			clientset,
			[]corev1.Node{*liveNode},
			func(context.Context, string) (map[types.NamespacedName]*VolumeStats, error) {
				return nil, metricsErr
			},
		)
		if !errors.Is(err, metricsErr) {
			t.Fatalf("collectPVCUsage returned %v, want %v", err, metricsErr)
		}
	})

	t.Run("fails closed when node deletion cannot be verified", func(t *testing.T) {
		clientset := fake.NewSimpleClientset()
		verificationErr := errors.New("node API unavailable")
		clientset.PrependReactor("get", "nodes", func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, verificationErr
		})
		_, err := collectPVCUsage(
			context.Background(),
			clientset,
			[]corev1.Node{{ObjectMeta: metav1.ObjectMeta{Name: "unknown-node"}}},
			func(context.Context, string) (map[types.NamespacedName]*VolumeStats, error) {
				return nil, metricsErr
			},
		)
		if !errors.Is(err, verificationErr) {
			t.Fatalf("collectPVCUsage returned %v, want %v", err, verificationErr)
		}
	})
}
