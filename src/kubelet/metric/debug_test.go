package metric

import (
	"testing"

	"github.com/newrelic/nri-kubernetes/v3/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDebugVolumeFiltering(t *testing.T) {
	// Create a config with filtering enabled
	cfg := &config.Kubelet{
		FilterSecretVolumes:    true,
		FilterConfigMapVolumes: true,
	}

	// Create a pod with a secret volume
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "my-secret-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "my-secret",
						},
					},
				},
			},
		},
	}

	// Test filtering
	result := shouldFilterVolumeByType("my-secret-volume", pod, cfg)

	t.Logf("Config: FilterSecretVolumes=%v, FilterConfigMapVolumes=%v",
		cfg.FilterSecretVolumes, cfg.FilterConfigMapVolumes)
	t.Logf("Pod has %d volumes", len(pod.Spec.Volumes))
	t.Logf("First volume name: %s", pod.Spec.Volumes[0].Name)
	t.Logf("First volume has Secret: %v", pod.Spec.Volumes[0].Secret != nil)
	t.Logf("Result: %v (should be true)", result)

	if !result {
		t.Errorf("Expected secret volume to be filtered, but it wasn't")
	}
}

func TestDebugConfigMapFiltering(t *testing.T) {
	// Create a config with filtering enabled
	cfg := &config.Kubelet{
		FilterSecretVolumes:    true,
		FilterConfigMapVolumes: true,
	}

	// Create a pod with a configmap volume
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "my-configmap-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "my-config",
							},
						},
					},
				},
			},
		},
	}

	// Test filtering
	result := shouldFilterVolumeByType("my-configmap-volume", pod, cfg)

	t.Logf("Config: FilterSecretVolumes=%v, FilterConfigMapVolumes=%v",
		cfg.FilterSecretVolumes, cfg.FilterConfigMapVolumes)
	t.Logf("Pod has %d volumes", len(pod.Spec.Volumes))
	t.Logf("First volume name: %s", pod.Spec.Volumes[0].Name)
	t.Logf("First volume has ConfigMap: %v", pod.Spec.Volumes[0].ConfigMap != nil)
	t.Logf("Result: %v (should be true)", result)

	if !result {
		t.Errorf("Expected configmap volume to be filtered, but it wasn't")
	}
}

func TestDebugFilteringDisabled(t *testing.T) {
	// Create a config with filtering disabled
	cfg := &config.Kubelet{
		FilterSecretVolumes:    false,
		FilterConfigMapVolumes: false,
	}

	// Create a pod with a secret volume
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "my-secret-volume",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "my-secret",
						},
					},
				},
			},
		},
	}

	// Test filtering
	result := shouldFilterVolumeByType("my-secret-volume", pod, cfg)

	t.Logf("Config: FilterSecretVolumes=%v, FilterConfigMapVolumes=%v",
		cfg.FilterSecretVolumes, cfg.FilterConfigMapVolumes)
	t.Logf("Result: %v (should be false since filtering is disabled)", result)

	if result {
		t.Errorf("Expected secret volume NOT to be filtered when config is disabled, but it was filtered")
	}
}

func TestDebugVolumeNotInSpec(t *testing.T) {
	// Create a config with filtering enabled
	cfg := &config.Kubelet{
		FilterSecretVolumes:    true,
		FilterConfigMapVolumes: true,
	}

	// Create a pod with a different volume
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
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
	}

	// Test filtering for a volume that doesn't exist
	result := shouldFilterVolumeByType("non-existent-volume", pod, cfg)

	t.Logf("Config: FilterSecretVolumes=%v, FilterConfigMapVolumes=%v",
		cfg.FilterSecretVolumes, cfg.FilterConfigMapVolumes)
	t.Logf("Pod has %d volumes", len(pod.Spec.Volumes))
	t.Logf("Looking for: non-existent-volume")
	t.Logf("Result: %v (should be false - volume not found)", result)

	if result {
		t.Errorf("Expected volume not in spec to NOT be filtered, but it was filtered")
	}
}

func TestDebugNilPodAndConfig(t *testing.T) {
	t.Run("nil pod", func(t *testing.T) {
		cfg := &config.Kubelet{
			FilterSecretVolumes: true,
		}
		result := shouldFilterVolumeByType("any-volume", nil, cfg)
		t.Logf("Result with nil pod: %v (should be false)", result)
		if result {
			t.Errorf("Expected false when pod is nil")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
		}
		result := shouldFilterVolumeByType("any-volume", pod, nil)
		t.Logf("Result with nil config: %v (should be false)", result)
		if result {
			t.Errorf("Expected false when config is nil")
		}
	})
}
