# Volume Filtering for Kubelet Scraper

## Overview

The New Relic Kubernetes integration now supports filtering of volume metrics scraped from the kubelet `/stats/summary` endpoint. This feature allows you to exclude filesystem and volume information for specific volume types, reducing data ingestion and focusing on relevant storage metrics.

## Filtered Volume Types

The integration can filter out the following volume types:

1. **Service Account Tokens** - Projected volumes containing Kubernetes service account tokens (typically named `kube-api-access-*`)
2. **Secrets** - Volumes mounted from Kubernetes Secrets
3. **ConfigMaps** - Volumes mounted from Kubernetes ConfigMaps

## Configuration

Volume filtering is configured in the `kubelet` section of the integration configuration. There are three configuration options:

### Configuration Options

```yaml
kubelet:
  # Always filters service account token volumes (enabled by default)
  # Volumes with names starting with "kube-api-access-" are automatically filtered
  filterServiceAccountVolumes: true
  
  # Filter volumes mounted from Secrets
  filterSecretVolumes: true
  
  # Filter volumes mounted from ConfigMaps  
  filterConfigMapVolumes: true
```

### Default Behavior

By default, **service account token volumes are always filtered** regardless of configuration. This is because these volumes:
- Are present in nearly every pod
- Contain minimal, predictable data
- Rarely need monitoring
- Can significantly increase data volume

The filtering happens by detecting volume names with the `kube-api-access-` prefix, which is the standard naming pattern for service account token projected volumes in Kubernetes.

### Type-Based Filtering

For more precise filtering of Secrets and ConfigMaps, the integration cross-references volume information from:
- `/stats/summary` endpoint (provides filesystem metrics)
- `/pods` endpoint (provides volume type information)

This allows accurate identification of volume types for filtering.

## How It Works

### Data Flow

1. **Kubelet Scraper** fetches pod specifications from `/pods` endpoint
2. **Stats Summary** is retrieved from `/stats/summary` endpoint containing volume metrics
3. **Volume Filtering** is applied:
   - Name-based filtering for service account tokens (always active)
   - Type-based filtering for secrets and configmaps (if enabled)
4. **Filtered Data** is processed and sent to New Relic

### Filtering Logic

```
For each volume in /stats/summary:
  
  1. Check if volume name starts with "kube-api-access-"
     → If yes, FILTER (skip this volume)
  
  2. If filterSecretVolumes is enabled:
     → Look up volume in pod spec
     → If volume type is Secret, FILTER
  
  3. If filterConfigMapVolumes is enabled:
     → Look up volume in pod spec  
     → If volume type is ConfigMap, FILTER
  
  4. If filterServiceAccountVolumes is enabled:
     → Check if projected volume contains ServiceAccountToken
     → If yes, FILTER
  
  5. Otherwise, INCLUDE the volume metrics
```

## Examples

### Example 1: Filter Only Service Account Tokens (Default)

By default, only service account token volumes are filtered:

```yaml
kubelet:
  enabled: true
  # No explicit filtering config needed
  # Service account tokens are filtered automatically
```

**Result:** Volumes like `kube-api-access-abc123` are filtered, but secrets and configmaps are still reported.

### Example 2: Filter All Sensitive Volumes

To filter all sensitive volume types:

```yaml
kubelet:
  enabled: true
  filterServiceAccountVolumes: true  # Explicitly enable (already default)
  filterSecretVolumes: true
  filterConfigMapVolumes: true
```

**Result:** Service account tokens, secrets, and configmaps are all filtered from volume metrics.

### Example 3: Filter Secrets Only

To filter only secret volumes while keeping configmaps:

```yaml
kubelet:
  enabled: true
  filterSecretVolumes: true
  filterConfigMapVolumes: false
```

**Result:** Secret volumes are filtered, but configmap volumes are still reported.

## Benefits

1. **Reduced Data Volume** - Eliminates metrics for volumes that rarely need monitoring
2. **Cost Savings** - Lower data ingestion reduces New Relic costs
3. **Cleaner Dashboards** - Focus on actual application storage without clutter from system volumes
4. **Security** - Less metadata about secret and configmap mounts is transmitted
5. **Performance** - Reduced processing overhead for filtered volumes

## Impact on Monitoring

### What You'll Still See

- **PersistentVolumes (PV)** - Still reported
- **PersistentVolumeClaims (PVC)** - Still reported  
- **EmptyDir volumes** - Still reported
- **HostPath volumes** - Still reported
- **Other volume types** - Still reported unless explicitly filtered

### What Gets Filtered

When filtering is enabled, you will no longer see volume metrics for:
- Service account token projected volumes (`kube-api-access-*`)
- Secret volumes (if `filterSecretVolumes: true`)
- ConfigMap volumes (if `filterConfigMapVolumes: true`)

This means no `K8sVolumeSample` metrics will be generated for these filtered volume types.

## Troubleshooting

### Volumes Still Appearing Despite Filtering

If volumes are still appearing after enabling filtering:

1. **Check configuration syntax** - Ensure YAML is properly formatted
2. **Restart the integration** - Configuration changes require pod restart
3. **Verify volume type** - Use `kubectl describe pod` to check actual volume type
4. **Check logs** - Look for warnings about pod spec retrieval failures

### Warning Messages

If you see warnings like:
```
Failed to get pod specs for volume filtering: <error>
```

This means type-based filtering (for secrets/configmaps) may not work, but name-based filtering (service account tokens) will still function.

Common causes:
- Insufficient RBAC permissions to read pods
- Network issues connecting to kubelet or API server
- Pod information not yet available

## Technical Details

### Implementation

- **Location**: `src/kubelet/metric/metric.go`
- **Function**: `GroupStatsSummaryWithConfig()`
- **Config**: `internal/config/config.go` (`Kubelet` struct)

### Performance

- Filtering happens during data collection, before metrics are sent to New Relic
- Pod specification lookup is cached per scrape cycle
- Minimal performance overhead (< 1ms per filtered volume)

### Compatibility

- **Kubernetes versions**: All supported versions (1.21+)
- **Integration versions**: 3.x and later
- **Backward compatible**: Default behavior unchanged (only service account tokens filtered)

## Related Configuration

This feature works alongside other kubelet configuration options:

```yaml
kubelet:
  enabled: true
  port: 10250
  scheme: https
  timeout: 10s
  
  # Volume filtering options
  filterServiceAccountVolumes: true
  filterSecretVolumes: true
  filterConfigMapVolumes: true
  
  # Other options remain unchanged
  fetchPodsFromKubeService: false
  scraperMaxReruns: 3
```

## Additional Resources

- [Kubelet API Documentation](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/)
- [Volume Types in Kubernetes](https://kubernetes.io/docs/concepts/storage/volumes/)
- [New Relic Kubernetes Integration](https://docs.newrelic.com/docs/kubernetes-pixie/kubernetes-integration/get-started/introduction-kubernetes-integration/)