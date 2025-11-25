package metric

import (
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/kubelet/pkg/apis/stats/v1alpha1"

	"github.com/newrelic/nri-kubernetes/v3/internal/config"
	"github.com/newrelic/nri-kubernetes/v3/src/definition"
)

// Helper functions for pointer types
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func TestGetAzureVolumeIdentifier_AzureFile(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "my-azure-file",
					VolumeSource: corev1.VolumeSource{
						AzureFile: &corev1.AzureFileVolumeSource{
							SecretName: "azure-secret",
							ShareName:  "logs-share",
							ReadOnly:   false,
						},
					},
				},
			},
		},
	}

	identifier := getAzureVolumeIdentifier("my-azure-file", pod)
	expected := "azurefile:default:azure-secret:logs-share"

	if identifier != expected {
		t.Errorf("Expected %s, got %s", expected, identifier)
	}
}

func TestGetAzureVolumeIdentifier_AzureFile_DifferentNamespace(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "production",
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "my-azure-file",
					VolumeSource: corev1.VolumeSource{
						AzureFile: &corev1.AzureFileVolumeSource{
							SecretName: "azure-secret",
							ShareName:  "logs-share",
							ReadOnly:   false,
						},
					},
				},
			},
		},
	}

	identifier := getAzureVolumeIdentifier("my-azure-file", pod)
	expected := "azurefile:production:azure-secret:logs-share"

	if identifier != expected {
		t.Errorf("Expected %s, got %s", expected, identifier)
	}
}

func TestGetAzureVolumeIdentifier_AzureDisk_WithDiskName(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "my-azure-disk",
					VolumeSource: corev1.VolumeSource{
						AzureDisk: &corev1.AzureDiskVolumeSource{
							DiskName:    "my-disk-vol",
							DataDiskURI: "/subscriptions/sub-id/resourceGroups/rg/providers/Microsoft.Compute/disks/my-disk-vol",
							FSType:      stringPtr("ext4"),
							ReadOnly:    boolPtr(false),
						},
					},
				},
			},
		},
	}

	identifier := getAzureVolumeIdentifier("my-azure-disk", pod)
	expected := "azuredisk:name:my-disk-vol"

	if identifier != expected {
		t.Errorf("Expected %s, got %s", expected, identifier)
	}
}

func TestGetAzureVolumeIdentifier_AzureDisk_WithURIOnly(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "my-azure-disk",
					VolumeSource: corev1.VolumeSource{
						AzureDisk: &corev1.AzureDiskVolumeSource{
							DiskName:    "",
							DataDiskURI: "/subscriptions/sub-id/resourceGroups/rg/providers/Microsoft.Compute/disks/my-disk-vol",
							FSType:      stringPtr("ext4"),
							ReadOnly:    boolPtr(false),
						},
					},
				},
			},
		},
	}

	identifier := getAzureVolumeIdentifier("my-azure-disk", pod)
	expected := "azuredisk:uri:/subscriptions/sub-id/resourceGroups/rg/providers/Microsoft.Compute/disks/my-disk-vol"

	if identifier != expected {
		t.Errorf("Expected %s, got %s", expected, identifier)
	}
}

func TestGetAzureVolumeIdentifier_NonAzure(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "emptydir-vol",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	identifier := getAzureVolumeIdentifier("emptydir-vol", pod)

	if identifier != "" {
		t.Errorf("Expected empty string for non-Azure volume, got %s", identifier)
	}
}

func TestGetAzureVolumeIdentifier_VolumeNotFound(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "some-volume",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	identifier := getAzureVolumeIdentifier("non-existent-volume", pod)

	if identifier != "" {
		t.Errorf("Expected empty string for non-existent volume, got %s", identifier)
	}
}

func TestGetAzureVolumeIdentifier_NilPod(t *testing.T) {
	identifier := getAzureVolumeIdentifier("any-volume", nil)

	if identifier != "" {
		t.Errorf("Expected empty string for nil pod, got %s", identifier)
	}
}

func TestEnrichAzureVolumeMetrics_AzureFile(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "my-azure-file",
					VolumeSource: corev1.VolumeSource{
						AzureFile: &corev1.AzureFileVolumeSource{
							SecretName: "azure-secret",
							ShareName:  "logs-share",
							ReadOnly:   true,
						},
					},
				},
			},
		},
	}

	metrics := make(definition.RawMetrics)
	enrichAzureVolumeMetrics(metrics, "my-azure-file", pod)

	if metrics["azureVolumeType"] != "azureFile" {
		t.Errorf("Expected azureVolumeType=azureFile, got %v", metrics["azureVolumeType"])
	}
	if metrics["azureShareName"] != "logs-share" {
		t.Errorf("Expected azureShareName=logs-share, got %v", metrics["azureShareName"])
	}
	if metrics["azureSecretName"] != "azure-secret" {
		t.Errorf("Expected azureSecretName=azure-secret, got %v", metrics["azureSecretName"])
	}
	if metrics["azureReadOnly"] != true {
		t.Errorf("Expected azureReadOnly=true, got %v", metrics["azureReadOnly"])
	}
}

func TestEnrichAzureVolumeMetrics_AzureDisk(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "my-azure-disk",
					VolumeSource: corev1.VolumeSource{
						AzureDisk: &corev1.AzureDiskVolumeSource{
							DiskName:    "my-disk-vol",
							DataDiskURI: "/subscriptions/sub-id/disks/my-disk-vol",
							FSType:      stringPtr("ext4"),
							ReadOnly:    boolPtr(false),
						},
					},
				},
			},
		},
	}

	metrics := make(definition.RawMetrics)
	enrichAzureVolumeMetrics(metrics, "my-azure-disk", pod)

	if metrics["azureVolumeType"] != "azureDisk" {
		t.Errorf("Expected azureVolumeType=azureDisk, got %v", metrics["azureVolumeType"])
	}
	if metrics["azureDiskName"] != "my-disk-vol" {
		t.Errorf("Expected azureDiskName=my-disk-vol, got %v", metrics["azureDiskName"])
	}
	if metrics["azureDiskURI"] != "/subscriptions/sub-id/disks/my-disk-vol" {
		t.Errorf("Expected azureDiskURI, got %v", metrics["azureDiskURI"])
	}
	if metrics["azureFSType"] != "ext4" {
		t.Errorf("Expected azureFSType=ext4, got %v", metrics["azureFSType"])
	}
	if metrics["azureReadOnly"] != false {
		t.Errorf("Expected azureReadOnly=false, got %v", metrics["azureReadOnly"])
	}
}

func TestEnrichAzureVolumeMetrics_NonAzure(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "emptydir-vol",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	metrics := make(definition.RawMetrics)
	enrichAzureVolumeMetrics(metrics, "emptydir-vol", pod)

	// Should not add any Azure-specific fields
	if _, exists := metrics["azureVolumeType"]; exists {
		t.Errorf("Should not add azureVolumeType for non-Azure volume")
	}
}

func TestEnrichAzureVolumeMetrics_NilPod(t *testing.T) {
	metrics := make(definition.RawMetrics)
	enrichAzureVolumeMetrics(metrics, "any-volume", nil)

	// Should not crash, should not add any fields
	if len(metrics) > 0 {
		t.Errorf("Expected no metrics for nil pod, got %d", len(metrics))
	}
}

func TestAzureDeduplication_SinglePodSingleVolume(t *testing.T) {
	pods := map[string]*corev1.Pod{
		"default_pod-1": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-1",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "azure-logs",
						VolumeSource: corev1.VolumeSource{
							AzureFile: &corev1.AzureFileVolumeSource{
								SecretName: "azure-secret",
								ShareName:  "application-logs",
								ReadOnly:   false,
							},
						},
					},
				},
			},
		},
	}

	statsSummary := &v1.Summary{
		Node: v1.NodeStats{NodeName: "test-node"},
		Pods: []v1.PodStats{
			{
				PodRef: v1.PodReference{Name: "pod-1", Namespace: "default"},
				VolumeStats: []v1.VolumeStats{
					{Name: "azure-logs", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(1000), CapacityBytes: uint64Ptr(5000)}},
				},
			},
		},
	}

	cfg := &config.Kubelet{
		DeduplicateAzureVolumes: true,
	}

	groups, errs := GroupStatsSummaryWithConfig(statsSummary, pods, cfg)

	if len(errs) > 0 {
		t.Fatalf("Unexpected errors: %v", errs)
	}

	volumeMetrics := groups["volume"]
	if len(volumeMetrics) != 1 {
		t.Errorf("Expected 1 volume metric, got %d", len(volumeMetrics))
	}
}

func TestAzureDeduplication_MultiplePodsSharedVolume(t *testing.T) {
	// Create 3 pods all mounting the same Azure File share
	pods := make(map[string]*corev1.Pod)

	for i := 1; i <= 3; i++ {
		podName := fmt.Sprintf("pod-%d", i)
		pods[fmt.Sprintf("default_%s", podName)] = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "shared-logs",
						VolumeSource: corev1.VolumeSource{
							AzureFile: &corev1.AzureFileVolumeSource{
								SecretName: "azure-secret",
								ShareName:  "application-logs", // SAME share for all pods
								ReadOnly:   false,
							},
						},
					},
				},
			},
		}
	}

	statsSummary := &v1.Summary{
		Node: v1.NodeStats{NodeName: "test-node"},
		Pods: []v1.PodStats{
			{
				PodRef: v1.PodReference{Name: "pod-1", Namespace: "default"},
				VolumeStats: []v1.VolumeStats{
					{Name: "shared-logs", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(1000)}},
				},
			},
			{
				PodRef: v1.PodReference{Name: "pod-2", Namespace: "default"},
				VolumeStats: []v1.VolumeStats{
					{Name: "shared-logs", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(1000)}},
				},
			},
			{
				PodRef: v1.PodReference{Name: "pod-3", Namespace: "default"},
				VolumeStats: []v1.VolumeStats{
					{Name: "shared-logs", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(1000)}},
				},
			},
		},
	}

	cfg := &config.Kubelet{
		DeduplicateAzureVolumes: true,
	}

	groups, errs := GroupStatsSummaryWithConfig(statsSummary, pods, cfg)

	if len(errs) > 0 {
		t.Fatalf("Unexpected errors: %v", errs)
	}

	// Should only have 1 volume metric (from first pod that has the share)
	volumeMetrics := groups["volume"]
	if len(volumeMetrics) != 1 {
		t.Errorf("Expected 1 volume metric (deduplicated), got %d", len(volumeMetrics))
	}

	// Verify the metric has Azure metadata
	var foundMetric definition.RawMetrics
	for _, metric := range volumeMetrics {
		foundMetric = metric
		break
	}

	if foundMetric["azureVolumeType"] != "azureFile" {
		t.Errorf("Expected azureVolumeType=azureFile in deduplicated metric")
	}
	if foundMetric["azureShareName"] != "application-logs" {
		t.Errorf("Expected azureShareName=application-logs in deduplicated metric")
	}
}

func TestAzureDeduplication_DifferentShares(t *testing.T) {
	// Create 2 pods with DIFFERENT Azure File shares
	pods := map[string]*corev1.Pod{
		"default_pod-1": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-1",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "logs-volume",
						VolumeSource: corev1.VolumeSource{
							AzureFile: &corev1.AzureFileVolumeSource{
								SecretName: "azure-secret",
								ShareName:  "logs-share",
								ReadOnly:   false,
							},
						},
					},
				},
			},
		},
		"default_pod-2": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-2",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "config-volume",
						VolumeSource: corev1.VolumeSource{
							AzureFile: &corev1.AzureFileVolumeSource{
								SecretName: "azure-secret",
								ShareName:  "config-share", // DIFFERENT share
								ReadOnly:   false,
							},
						},
					},
				},
			},
		},
	}

	statsSummary := &v1.Summary{
		Node: v1.NodeStats{NodeName: "test-node"},
		Pods: []v1.PodStats{
			{
				PodRef: v1.PodReference{Name: "pod-1", Namespace: "default"},
				VolumeStats: []v1.VolumeStats{
					{Name: "logs-volume", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(1000)}},
				},
			},
			{
				PodRef: v1.PodReference{Name: "pod-2", Namespace: "default"},
				VolumeStats: []v1.VolumeStats{
					{Name: "config-volume", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(2000)}},
				},
			},
		},
	}

	cfg := &config.Kubelet{
		DeduplicateAzureVolumes: true,
	}

	groups, errs := GroupStatsSummaryWithConfig(statsSummary, pods, cfg)

	if len(errs) > 0 {
		t.Fatalf("Unexpected errors: %v", errs)
	}

	// Should have 2 volume metrics (different shares)
	volumeMetrics := groups["volume"]
	if len(volumeMetrics) != 2 {
		t.Errorf("Expected 2 volume metrics (different shares), got %d", len(volumeMetrics))
	}
}

func TestAzureDeduplication_MixedAzureAndNonAzure(t *testing.T) {
	// Pod 1: Azure File + EmptyDir
	// Pod 2: Same Azure File + ConfigMap
	pods := map[string]*corev1.Pod{
		"default_pod-1": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-1",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "shared-logs",
						VolumeSource: corev1.VolumeSource{
							AzureFile: &corev1.AzureFileVolumeSource{
								SecretName: "azure-secret",
								ShareName:  "application-logs",
								ReadOnly:   false,
							},
						},
					},
					{
						Name: "cache",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		},
		"default_pod-2": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-2",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "shared-logs",
						VolumeSource: corev1.VolumeSource{
							AzureFile: &corev1.AzureFileVolumeSource{
								SecretName: "azure-secret",
								ShareName:  "application-logs", // SAME share
								ReadOnly:   false,
							},
						},
					},
					{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "app-config",
								},
							},
						},
					},
				},
			},
		},
	}

	statsSummary := &v1.Summary{
		Node: v1.NodeStats{NodeName: "test-node"},
		Pods: []v1.PodStats{
			{
				PodRef: v1.PodReference{Name: "pod-1", Namespace: "default"},
				VolumeStats: []v1.VolumeStats{
					{Name: "shared-logs", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(1000)}},
					{Name: "cache", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(500)}},
				},
			},
			{
				PodRef: v1.PodReference{Name: "pod-2", Namespace: "default"},
				VolumeStats: []v1.VolumeStats{
					{Name: "shared-logs", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(1000)}},
					{Name: "config", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(100)}},
				},
			},
		},
	}

	cfg := &config.Kubelet{
		DeduplicateAzureVolumes: true,
	}

	groups, errs := GroupStatsSummaryWithConfig(statsSummary, pods, cfg)

	if len(errs) > 0 {
		t.Fatalf("Unexpected errors: %v", errs)
	}

	// Should have 3 volume metrics:
	// - shared-logs (Azure) from pod-1 only (deduplicated)
	// - cache (EmptyDir) from pod-1
	// - config (ConfigMap) from pod-2
	volumeMetrics := groups["volume"]
	if len(volumeMetrics) != 3 {
		t.Errorf("Expected 3 volume metrics (1 Azure deduplicated + 2 non-Azure), got %d", len(volumeMetrics))
	}
}

func TestAzureDeduplication_Disabled(t *testing.T) {
	// Create 3 pods all mounting the same Azure File share
	pods := make(map[string]*corev1.Pod)

	for i := 1; i <= 3; i++ {
		podName := fmt.Sprintf("pod-%d", i)
		pods[fmt.Sprintf("default_%s", podName)] = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "shared-logs",
						VolumeSource: corev1.VolumeSource{
							AzureFile: &corev1.AzureFileVolumeSource{
								SecretName: "azure-secret",
								ShareName:  "application-logs",
								ReadOnly:   false,
							},
						},
					},
				},
			},
		}
	}

	statsSummary := &v1.Summary{
		Node: v1.NodeStats{NodeName: "test-node"},
		Pods: []v1.PodStats{
			{
				PodRef: v1.PodReference{Name: "pod-1", Namespace: "default"},
				VolumeStats: []v1.VolumeStats{
					{Name: "shared-logs", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(1000)}},
				},
			},
			{
				PodRef: v1.PodReference{Name: "pod-2", Namespace: "default"},
				VolumeStats: []v1.VolumeStats{
					{Name: "shared-logs", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(1000)}},
				},
			},
			{
				PodRef: v1.PodReference{Name: "pod-3", Namespace: "default"},
				VolumeStats: []v1.VolumeStats{
					{Name: "shared-logs", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(1000)}},
				},
			},
		},
	}

	cfg := &config.Kubelet{
		DeduplicateAzureVolumes: false, // DISABLED
	}

	groups, errs := GroupStatsSummaryWithConfig(statsSummary, pods, cfg)

	if len(errs) > 0 {
		t.Fatalf("Unexpected errors: %v", errs)
	}

	// Should have 3 volume metrics (no deduplication)
	volumeMetrics := groups["volume"]
	if len(volumeMetrics) != 3 {
		t.Errorf("Expected 3 volume metrics (deduplication disabled), got %d", len(volumeMetrics))
	}
}

func TestAzureDeduplication_DifferentNamespacesSameShare(t *testing.T) {
	// Two pods in different namespaces with "same" share name
	// Should be treated as DIFFERENT shares (different secrets)
	pods := map[string]*corev1.Pod{
		"namespace1_pod-1": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-1",
				Namespace: "namespace1",
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "logs",
						VolumeSource: corev1.VolumeSource{
							AzureFile: &corev1.AzureFileVolumeSource{
								SecretName: "azure-secret",
								ShareName:  "logs-share",
								ReadOnly:   false,
							},
						},
					},
				},
			},
		},
		"namespace2_pod-2": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-2",
				Namespace: "namespace2",
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "logs",
						VolumeSource: corev1.VolumeSource{
							AzureFile: &corev1.AzureFileVolumeSource{
								SecretName: "azure-secret",
								ShareName:  "logs-share", // Same name, different namespace
								ReadOnly:   false,
							},
						},
					},
				},
			},
		},
	}

	statsSummary := &v1.Summary{
		Node: v1.NodeStats{NodeName: "test-node"},
		Pods: []v1.PodStats{
			{
				PodRef: v1.PodReference{Name: "pod-1", Namespace: "namespace1"},
				VolumeStats: []v1.VolumeStats{
					{Name: "logs", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(1000)}},
				},
			},
			{
				PodRef: v1.PodReference{Name: "pod-2", Namespace: "namespace2"},
				VolumeStats: []v1.VolumeStats{
					{Name: "logs", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(2000)}},
				},
			},
		},
	}

	cfg := &config.Kubelet{
		DeduplicateAzureVolumes: true,
	}

	groups, errs := GroupStatsSummaryWithConfig(statsSummary, pods, cfg)

	if len(errs) > 0 {
		t.Fatalf("Unexpected errors: %v", errs)
	}

	// Should have 2 volume metrics (different namespaces = different shares)
	volumeMetrics := groups["volume"]
	if len(volumeMetrics) != 2 {
		t.Errorf("Expected 2 volume metrics (different namespaces), got %d", len(volumeMetrics))
	}
}

func TestAzureDeduplication_AzureDisk(t *testing.T) {
	// Azure Disk with same diskName in two pods
	// (Note: This is unlikely in practice as disks are typically RWO, but testing the logic)
	pods := map[string]*corev1.Pod{
		"default_pod-1": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-1",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "data-disk",
						VolumeSource: corev1.VolumeSource{
							AzureDisk: &corev1.AzureDiskVolumeSource{
								DiskName:    "my-shared-disk",
								DataDiskURI: "/subscriptions/sub-id/disks/my-shared-disk",
								FSType:      stringPtr("ext4"),
								ReadOnly:    boolPtr(false),
							},
						},
					},
				},
			},
		},
		"default_pod-2": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-2",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "data-disk",
						VolumeSource: corev1.VolumeSource{
							AzureDisk: &corev1.AzureDiskVolumeSource{
								DiskName:    "my-shared-disk", // SAME disk
								DataDiskURI: "/subscriptions/sub-id/disks/my-shared-disk",
								FSType:      stringPtr("ext4"),
								ReadOnly:    boolPtr(false),
							},
						},
					},
				},
			},
		},
	}

	statsSummary := &v1.Summary{
		Node: v1.NodeStats{NodeName: "test-node"},
		Pods: []v1.PodStats{
			{
				PodRef: v1.PodReference{Name: "pod-1", Namespace: "default"},
				VolumeStats: []v1.VolumeStats{
					{Name: "data-disk", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(5000)}},
				},
			},
			{
				PodRef: v1.PodReference{Name: "pod-2", Namespace: "default"},
				VolumeStats: []v1.VolumeStats{
					{Name: "data-disk", FsStats: v1.FsStats{AvailableBytes: uint64Ptr(5000)}},
				},
			},
		},
	}

	cfg := &config.Kubelet{
		DeduplicateAzureVolumes: true,
	}

	groups, errs := GroupStatsSummaryWithConfig(statsSummary, pods, cfg)

	if len(errs) > 0 {
		t.Fatalf("Unexpected errors: %v", errs)
	}

	// Should only have 1 volume metric (same disk, deduplicated)
	volumeMetrics := groups["volume"]
	if len(volumeMetrics) != 1 {
		t.Errorf("Expected 1 volume metric (deduplicated Azure Disk), got %d", len(volumeMetrics))
	}

	// Verify the metric has Azure Disk metadata
	var foundMetric definition.RawMetrics
	for _, metric := range volumeMetrics {
		foundMetric = metric
		break
	}

	if foundMetric["azureVolumeType"] != "azureDisk" {
		t.Errorf("Expected azureVolumeType=azureDisk in deduplicated metric")
	}
	if foundMetric["azureDiskName"] != "my-shared-disk" {
		t.Errorf("Expected azureDiskName=my-shared-disk in deduplicated metric")
	}
}
