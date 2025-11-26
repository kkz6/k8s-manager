# GKE Cluster Downscaling Documentation

## Date
2025-10-16

## Objective
Scale down the Laravel GKE cluster from 5 nodes with 2 HTTP replicas to 1 node with 1 HTTP replica, without using a LoadBalancer.

## Initial State

### Cluster Configuration
- **Cluster Name**: laravel-cluster-stg
- **Region**: us-central1 (USA - Iowa)
- **Zones**: us-central1-a, us-central1-b, us-central1-f (3 zones)
- **Total Nodes**: 5 nodes
- **Machine Type**: e2-medium (2 vCPUs, 4GB RAM)
- **Node Pool**: laravel-nodes-stg
- **Autoscaling**: Enabled (min: 1, max: 2 nodes per zone)

### Deployment Configuration
- **laravel-http**: 2 replicas
- **laravel-horizon**: 2 replicas
- **laravel-scheduler**: 1 replica

### Service Configuration
- **laravel-http-service**: NodePort (already configured, no LoadBalancer)
- **Port**: 80:30808/TCP

### HPA Configuration
- **laravel-http**: minPods: 2, maxPods: 4, target CPU: 70%
- **laravel-horizon**: minPods: 1, maxPods: 2, target CPU: 80%, memory: 85%

## Steps Performed

### 1. Cluster Discovery
```bash
# List all clusters
gcloud container clusters list

# List node pools
gcloud container node-pools list --cluster=laravel-cluster-stg --region=us-central1

# Check node pool details
gcloud container node-pools describe laravel-nodes-stg --cluster=laravel-cluster-stg --region=us-central1

# Check services
kubectl get services --all-namespaces -o wide

# Check deployments
kubectl get deployments --all-namespaces -o wide
```

### 2. Scale Down Deployments
```bash
# Scale laravel-http from 2 to 1 replica
kubectl scale deployment laravel-http --replicas=1 -n laravel-app

# Scale laravel-horizon from 2 to 1 replica
kubectl scale deployment laravel-horizon --replicas=1 -n laravel-app
```

### 3. Update HPA Minimum Replicas
```bash
# Update laravel-http HPA minimum from 2 to 1
kubectl patch hpa laravel-http -n laravel-app --patch '{"spec":{"minReplicas":1}}'

# Verify HPA changes
kubectl get hpa -n laravel-app
```

### 4. Disable Node Pool Autoscaling
```bash
# Disable autoscaling on the node pool
gcloud container clusters update laravel-cluster-stg \
  --no-enable-autoscaling \
  --node-pool=laravel-nodes-stg \
  --region=us-central1 \
  --quiet
```

### 5. Reduce to Single Zone
```bash
# Update cluster to use only us-central1-a zone
gcloud container clusters update laravel-cluster-stg \
  --region=us-central1 \
  --node-locations=us-central1-a \
  --quiet
```

### 6. Scale Down Node Pool
```bash
# Resize node pool to 1 node per zone (1 total since single zone)
gcloud container clusters resize laravel-cluster-stg \
  --node-pool=laravel-nodes-stg \
  --num-nodes=1 \
  --region=us-central1 \
  --quiet
```

### 7. Upgrade Machine Type
**Issue**: The e2-medium nodes had insufficient CPU (100% allocated at 940m) for all pods after scaling down.

**Solution**: Recreate node pool with e2-standard-2 machine type.

```bash
# Delete existing node pool
gcloud container node-pools delete laravel-nodes-stg \
  --cluster=laravel-cluster-stg \
  --region=us-central1 \
  --quiet

# Create new node pool with larger machine type
gcloud container node-pools create laravel-nodes-stg \
  --cluster=laravel-cluster-stg \
  --region=us-central1 \
  --machine-type=e2-standard-2 \
  --num-nodes=1 \
  --node-locations=us-central1-a \
  --disk-size=30 \
  --no-enable-autoscaling \
  --quiet
```

## Final State

### Cluster Configuration
- **Cluster Name**: laravel-cluster-stg
- **Region**: us-central1
- **Zones**: us-central1-a (single zone)
- **Total Nodes**: 1 node
- **Machine Type**: e2-standard-2 (2 vCPUs, 8GB RAM)
- **Node Pool**: laravel-nodes-stg
- **Autoscaling**: Disabled

### Deployment Configuration
- **laravel-http**: 1 replica
- **laravel-horizon**: 1 replica
- **laravel-scheduler**: 1 replica

### Service Configuration
- **laravel-http-service**: NodePort (no LoadBalancer)
- **Port**: 80:30808/TCP
- **External Access**: Via NodePort on node external IP (34.9.45.254:30808)

### HPA Configuration
- **laravel-http**: minPods: 1, maxPods: 4, target CPU: 70%
- **laravel-horizon**: minPods: 1, maxPods: 2, target CPU: 80%, memory: 85%

### Resource Allocation (e2-standard-2 node)
- **CPU Requests**: ~940m allocated (system pods) + 750m (Laravel pods) = ~1690m total available
- **Memory**: Sufficient capacity for all pods

## Cost Savings
- **Node Count**: 5 → 1 (80% reduction)
- **Machine Type**: e2-medium → e2-standard-2 (slightly higher cost per node, but overall much cheaper)
- **Estimated Savings**: ~70-75% cost reduction

## Verification Commands

### Check Cluster Status
```bash
# View cluster summary
gcloud container clusters list

# View node details
kubectl get nodes -o wide

# Check all deployments
kubectl get deployments -n laravel-app

# Check all pods
kubectl get pods -n laravel-app

# Check services
kubectl get svc -n laravel-app

# Check HPA status
kubectl get hpa -n laravel-app

# View complete cluster info
kubectl get all -n laravel-app
```

### Access the Application
The application is accessible via NodePort:
```
http://<NODE_EXTERNAL_IP>:30808
```

Current node external IP: `34.9.45.254`

## Issues Encountered and Solutions

### Issue 1: HPA Scaling Back Up
**Problem**: After manually scaling down to 1 replica, the HPA scaled it back to 2 because minReplicas was set to 2.

**Solution**: Updated the HPA minReplicas from 2 to 1 using kubectl patch.

### Issue 2: Node Autoscaling
**Problem**: Cluster had autoscaling enabled (min: 1, max: 2 per zone), causing it to maintain more nodes than desired.

**Solution**: Disabled autoscaling completely on the node pool.

### Issue 3: Regional Cluster with Multiple Zones
**Problem**: Setting `--num-nodes=1` on a regional cluster created 1 node per zone (3 total).

**Solution**: Reduced cluster to single zone (us-central1-a) before resizing.

### Issue 4: Insufficient CPU
**Problem**: e2-medium nodes only had ~940m CPU available after system pods, but needed 750m for Laravel pods (3 pods × 250m each).

**Solution**: Upgraded to e2-standard-2 machine type with 2 vCPUs (~1940m allocatable).

## Future Considerations

### Moving to Japan Region
The cluster is currently in `us-central1` (USA). To move to a Japan region like `asia-northeast1` (Tokyo):

1. You cannot change the region of an existing cluster
2. Must create a new cluster in the desired region
3. Available Japan regions:
   - `asia-northeast1` (Tokyo) - zones: a, b, c
   - `asia-northeast2` (Osaka) - zones: a, b, c

#### Steps to Migrate to Japan
```bash
# 1. Create new cluster in Tokyo
gcloud container clusters create laravel-cluster-prod \
  --region=asia-northeast1 \
  --node-locations=asia-northeast1-a \
  --machine-type=e2-standard-2 \
  --num-nodes=1 \
  --disk-size=30

# 2. Deploy applications to new cluster
kubectl apply -f <your-k8s-manifests>

# 3. Migrate data/state if needed

# 4. Update DNS to point to new cluster

# 5. Delete old cluster after verification
gcloud container clusters delete laravel-cluster-stg --region=us-central1
```

### Scaling Back Up
If you need to scale back up:

```bash
# Enable autoscaling
gcloud container clusters update laravel-cluster-stg \
  --enable-autoscaling \
  --node-pool=laravel-nodes-stg \
  --min-nodes=1 \
  --max-nodes=3 \
  --region=us-central1

# Scale deployments
kubectl scale deployment laravel-http --replicas=2 -n laravel-app
kubectl scale deployment laravel-horizon --replicas=2 -n laravel-app

# Update HPA
kubectl patch hpa laravel-http -n laravel-app --patch '{"spec":{"minReplicas":2}}'

# Add more zones
gcloud container clusters update laravel-cluster-stg \
  --region=us-central1 \
  --node-locations=us-central1-a,us-central1-b,us-central1-f
```

## Notes
- All changes were made using gcloud CLI and kubectl commands
- No LoadBalancer service was used (already configured as NodePort)
- The scheduler deployment remained at 1 replica (already optimal)
- All pods are running successfully on the single node
- Health checks are passing for all deployments
