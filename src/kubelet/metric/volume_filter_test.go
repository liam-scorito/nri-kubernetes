package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubelet/pkg/apis/stats/v1alpha1"

	"github.com/newrelic/nri-kubernetes/v3/internal/config"
)

func TestShouldFilterVolume(t *testing.T) {
	tests := []struct {
		name       string
		volumeName string
		expected   bool
	}{
		{
			name:       "filters service account token volume",
			volumeName: "kube-api-access-abc123",
			expected:   true,
		},
		{
			name:       "filters another service account token volume",
			volumeName: "kube-api-access-xyz789",
			expected:   true,
		},
		{
			name:       "does not filter regular volume",
			volumeName: "my-data-volume",
			expected:   false,
		},
		{
			name:       "does not filter emptyDir volume",
			volumeName: "cache",
			expected:   false,
		},
		{
			name:       "does not filter pvc volume",
			volumeName: "storage",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldFilterVolume(tt.volumeName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldFilterVolumeByType(t *testing.T) {
	tests := []struct {
		name       string
		volumeName string
		pod        *corev1.Pod
		config     *config.Kubelet
		expected   bool
	}{
		{
			name:       "filters secret volume when FilterSecretVolumes is true",
			volumeName: "my-secret",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "my-secret",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "my-secret",
								},
							},
						},
					},
				},
			},
			config: &config.Kubelet{
				FilterSecretVolumes: true,
			},
			expected: true,
		},
		{
			name:       "does not filter secret volume when FilterSecretVolumes is false",
			volumeName: "my-secret",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "my-secret",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "my-secret",
								},
							},
						},
					},
				},
			},
			config: &config.Kubelet{
				FilterSecretVolumes: false,
			},
			expected: false,
		},
		{
			name:       "filters configmap volume when FilterConfigMapVolumes is true",
			volumeName: "my-configmap",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "my-configmap",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "my-configmap",
									},
								},
							},
						},
					},
				},
			},
			config: &config.Kubelet{
				FilterConfigMapVolumes: true,
			},
			expected: true,
		},
		{
			name:       "filters projected volume with service account token",
			volumeName: "kube-api-access-xyz",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "kube-api-access-xyz",
							VolumeSource: corev1.VolumeSource{
								Projected: &corev1.ProjectedVolumeSource{
									Sources: []corev1.VolumeProjection{
										{
											ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
												Path: "token",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			config: &config.Kubelet{
				FilterServiceAccountVolumes: true,
			},
			expected: true,
		},
		{
			name:       "filters projected volume with configmap",
			volumeName: "projected-volume",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "projected-volume",
							VolumeSource: corev1.VolumeSource{
								Projected: &corev1.ProjectedVolumeSource{
									Sources: []corev1.VolumeProjection{
										{
											ConfigMap: &corev1.ConfigMapProjection{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "my-config",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			config: &config.Kubelet{
				FilterConfigMapVolumes: true,
			},
			expected: true,
		},
		{
			name:       "does not filter emptyDir volume",
			volumeName: "cache",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "cache",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
			config: &config.Kubelet{
				FilterSecretVolumes:    true,
				FilterConfigMapVolumes: true,
			},
			expected: false,
		},
		{
			name:       "does not filter PVC volume",
			volumeName: "storage",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "storage",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "my-pvc",
								},
							},
						},
					},
				},
			},
			config: &config.Kubelet{
				FilterSecretVolumes:    true,
				FilterConfigMapVolumes: true,
			},
			expected: false,
		},
		{
			name:       "returns false when pod is nil",
			volumeName: "any-volume",
			pod:        nil,
			config: &config.Kubelet{
				FilterSecretVolumes: true,
			},
			expected: false,
		},
		{
			name:       "returns false when config is nil",
			volumeName: "any-volume",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "any-volume",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{},
							},
						},
					},
				},
			},
			config:   nil,
			expected: false,
		},
		{
			name:       "returns false for volume not in pod spec",
			volumeName: "non-existent-volume",
			pod: &corev1.Pod{
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "other-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
			config: &config.Kubelet{
				FilterSecretVolumes: true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldFilterVolumeByType(tt.volumeName, tt.pod, tt.config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGroupStatsSummaryWithConfig_FiltersVolumes(t *testing.T) {
	// Create a summary with service account token volumes
	summary := &v1alpha1.Summary{
		Node: v1alpha1.NodeStats{
			NodeName: "test-node",
		},
		Pods: []v1alpha1.PodStats{
			{
				PodRef: v1alpha1.PodReference{
					Name:      "test-pod",
					Namespace: "default",
				},
				VolumeStats: []v1alpha1.VolumeStats{
					{
						Name: "kube-api-access-abc123",
						FsStats: v1alpha1.FsStats{
							AvailableBytes: uint64Ptr(1000000),
							CapacityBytes:  uint64Ptr(2000000),
							UsedBytes:      uint64Ptr(1000000),
						},
					},
					{
						Name: "regular-volume",
						FsStats: v1alpha1.FsStats{
							AvailableBytes: uint64Ptr(5000000),
							CapacityBytes:  uint64Ptr(10000000),
							UsedBytes:      uint64Ptr(5000000),
						},
					},
				},
			},
		},
	}

	// Test without config (default behavior - should filter service account tokens)
	rawGroups, errs := GroupStatsSummary(summary)
	assert.Empty(t, errs)
	assert.NotNil(t, rawGroups["volume"])

	// Should only have the regular volume, not the service account token
	assert.Len(t, rawGroups["volume"], 1)
	_, hasRegular := rawGroups["volume"]["default_test-pod_regular-volume"]
	assert.True(t, hasRegular, "should have regular volume")
	_, hasToken := rawGroups["volume"]["default_test-pod_kube-api-access-abc123"]
	assert.False(t, hasToken, "should not have service account token volume")
}

func TestGroupStatsSummaryWithConfig_FiltersSecretVolumes(t *testing.T) {
	summary := &v1alpha1.Summary{
		Node: v1alpha1.NodeStats{
			NodeName: "test-node",
		},
		Pods: []v1alpha1.PodStats{
			{
				PodRef: v1alpha1.PodReference{
					Name:      "test-pod",
					Namespace: "default",
				},
				VolumeStats: []v1alpha1.VolumeStats{
					{
						Name: "my-secret",
						FsStats: v1alpha1.FsStats{
							AvailableBytes: uint64Ptr(1000000),
							CapacityBytes:  uint64Ptr(2000000),
							UsedBytes:      uint64Ptr(1000000),
						},
					},
					{
						Name: "regular-volume",
						FsStats: v1alpha1.FsStats{
							AvailableBytes: uint64Ptr(5000000),
							CapacityBytes:  uint64Ptr(10000000),
							UsedBytes:      uint64Ptr(5000000),
						},
					},
				},
			},
		},
	}

	podSpecs := map[string]*corev1.Pod{
		"default_test-pod": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Volumes: []corev1.Volume{
					{
						Name: "my-secret",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "my-secret",
							},
						},
					},
					{
						Name: "regular-volume",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		},
	}

	cfg := &config.Kubelet{
		FilterSecretVolumes: true,
	}

	rawGroups, errs := GroupStatsSummaryWithConfig(summary, podSpecs, cfg)
	assert.Empty(t, errs)
	assert.NotNil(t, rawGroups["volume"])

	// Should only have the regular volume, not the secret
	assert.Len(t, rawGroups["volume"], 1)
	_, hasRegular := rawGroups["volume"]["default_test-pod_regular-volume"]
	assert.True(t, hasRegular, "should have regular volume")
	_, hasSecret := rawGroups["volume"]["default_test-pod_my-secret"]
	assert.False(t, hasSecret, "should not have secret volume")
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}
