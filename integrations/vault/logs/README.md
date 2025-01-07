
# Configuring Vault for Dev Mode and Audit Log Ingestion into Apica Ascent

## Step 1: Create a Namespace for Vault
Create a namespace to isolate Vault resources.

```bash
kubectl create namespace vault
kubectl get namespace
```

## Step 2: Create a Persistent Volume Claim (PVC)
A Persistent Volume Claim (PVC) is required for Vault to store data persistently.

```bash
kubectl apply -f vault-pvc.yaml -n vault
kubectl get pvc -n vault
```

### Common Error:
If you encounter the error:
```plaintext
failed to provision volume with StorageClass "oci-bv": rpc error: code = InvalidArgument desc = invalid volume capabilities requested. Only SINGLE_NODE_WRITER is supported ('accessModes.ReadWriteOnce' on Kubernetes)
```
Follow these steps:
1. Proceed with the `ReadWriteOnce` mode.
2. Once the pods are running, update the Persistent Volume (PV) to `ReadWriteMany`:

```bash
kubectl get pv -n vault
kubectl edit pv/<PV_NAME> -n vault
```

Update the access mode in the PV YAML:
```yaml
spec:
  accessModes:
  - ReadWriteMany
```

## Step 3: Install Vault in Dev Mode
Deploy Vault in Dev Mode using Helm, enabling the UI and exposing it as a LoadBalancer.

```bash
helm upgrade --install vault hashicorp/vault        --set='server.dev.enabled=true'        --set='ui.enabled=true'        --set='ui.serviceType=LoadBalancer'        --namespace vault -f server.yaml --debug
```

Verify the installation:
```bash
kubectl get all -n vault
```

## Step 4: Enable Audit Logs
Audit logs track all Vault requests. Enable this feature in the Vault pod.

1. Access the Vault pod:
   ```bash
   kubectl exec -it pod/vault-0 -n vault -- sh
   ```
2. Log in to Vault:
   ```bash
   vault login root
   ```
3. Enable file-based audit logging:
   ```bash
   vault audit enable file file_path=/vault/logs/vault-audit.log log_format=json
   ```
4. Verify the audit configuration:
   ```bash
   vault audit list
   ```

## Step 5: Install Fluent Bit
Fluent Bit is used to forward audit logs to Apica Ascent.

1. Add the Fluent Bit Helm repository:
   ```bash
   helm repo add fluent https://fluent.github.io/helm-charts
   helm repo update
   ```

## Step 6: Configure Fluent Bit
### Create a ConfigMap:
The ConfigMap contains Fluent Bit configurations. Update the following placeholders in `fb-configmap.yaml`:
- `#TODO: REPLACE PATH`: Replace with the Vault audit log file path (e.g., `/vault/logs/vault-audit.log`).
- `#TODO: REPLACE HOST`: Replace with the Apica Ascent hostname or endpoint.
- `#TODO: REPLACE URI IF NEEDED`: Update the URI path if necessary.
- `#TODO: REPLACE Bearer token`: Replace with the Apica Ascent Bearer token.

Apply the ConfigMap:
```bash
kubectl apply -f fb-configmap.yaml -n vault
kubectl get configmap -n vault
```

### Deploy Fluent Bit:
```bash
kubectl apply -f fb-deployment.yaml -n vault
```

## Step 7: Generate Logs and Verify Fluent Bit Output
1. Log in to the Vault UI using the LoadBalancer IP or DNS name.
2. Perform actions like creating secrets to generate logs.

### Verify Fluent Bit Logs:
```bash
kubectl logs pod/<FLUENT_BIT_POD_NAME> -n vault
```
Ensure logs show successful processing of Vault audit logs.

## Step 8: Verify Logs in Apica Ascent
1. Log in to the Apica Ascent portal.
2. Navigate to **Logs & Insights**.
3. Locate the `vault-logs` namespace.
4. Open the Vault app to view the ingested logs.