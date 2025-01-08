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
## Step 3: Update the PVC accessmode to ReadWriteMany
-- Get PVC for the namespace vault
```bash
kubectl get pvc -n vault
```
-- Edit the PVC and update the access mode
```bash
kubectl edit pvc/<PVC_NAME> -n vault
```
```yaml
spec:
  accessModes:
  - ReadWriteMany
```

## Step 4: Install Vault in Dev Mode
Deploy Vault in Dev Mode using Helm, enabling the UI and exposing it as a LoadBalancer.

```bash
helm upgrade --install vault hashicorp/vault \
  --set='server.dev.enabled=true' \
  --set='ui.enabled=true' \
  --set='ui.serviceType=LoadBalancer' \
  --namespace vault -f server.yaml --debug
```

-- Verify the installation, and veriy the pods are running.
```bash
kubectl get all -n vault
```
-- Describe the verify the pods incase of any errors.
```bash
kubectl describe pod/<vault pod name> -nvault
```

## Step 4: Affter setting vault enable audit logs
-- Follow these steps to enable the audit logs track all Vault requests, by exec into the vault pod.

1. Access the Vault pod:
   ```bash
   kubectl exec -it pod/vault-0 -n vault -- sh
   ```
2. Log in to Vault as root:
   ```bash
   vault login root
   ```
3. Verify forthe existing audit configuration should display all the audit list:
   ```bash
   vault audit list
   ```
4. Enable file-based audit logging:
   ```bash
   vault audit enable file file_path=/vault/logs/vault-audit.log log_format=json
   ```
5. Verify the audit configuration:
   ```bash
   vault audit list
   ```

## Step 5: Install Fluent Bit to scpare the audit logs from the vault and ingest it to Apica Ascent.

1. Add the Fluent Bit Helm repository:
   ```bash
   helm repo add fluent https://fluent.github.io/helm-charts
   helm repo update
   ```
2. Create a ConfigMap for fluentBit:
-- Before applying the configmap, please update the 4 fields based on the configuration in `fb-configmap.yaml` file.
- [INPUT]: Replace `path` with the Vault audit log file path (e.g., `/vault/logs/vault-audit.log`).
- [OUTPUT]: Replace `Host` with the Apica Ascent hostname or endpoint.
- [OUTPUT]: Update `URI` path if necessary.
- [OUTPUT]: Update `Header` Authorization Bearer token.

3. Apply the ConfigMap:
```bash
kubectl apply -f fb-configmap.yaml -n vault
kubectl get configmap -n vault
```

4. Deploy Fluent Bit:
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



