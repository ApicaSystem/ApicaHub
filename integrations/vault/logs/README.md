# Configuring Vault for Dev Mode and Audit Log Ingestion into Apica Ascent

## Step 1: Create a Namespace for Vault
Create a namespace named `vault`:
```bash
kubectl create namespace vault
kubectl get namespace
```

## Step 2: Create a Persistent Volume Claim (PVC) with AccessMode `ReadWriteOnce`
Apply the PVC configuration for the Vault namespace:
```bash
kubectl apply -f vault-pvc.yaml -n vault
kubectl get pvc -n vault
```

### Common Error:
If you encounter the following error:
```plaintext
failed to provision volume with StorageClass "oci-bv": rpc error: code = InvalidArgument desc = invalid volume capabilities requested. Only SINGLE_NODE_WRITER is supported ('accessModes.ReadWriteOnce' on Kubernetes)
```
You can proceed to update the Persistent Volume (PV) later to `ReadWriteMany` mode after the pods are in the running state.

## Step 3: Install Vault in Dev Mode and Verify the Installation
Install Vault in dev mode with Helm and enable the UI:
```bash
helm upgrade --install vault hashicorp/vault \
       --set='server.dev.enabled=true' \
       --set='ui.enabled=true' \
       --set='ui.serviceType=LoadBalancer' \
       --namespace vault -f server.yaml --debug

kubectl get all -n vault
```

Verify the pods are in the running state, then edit the PV to change the access mode:
```bash
kubectl get pv -n vault
kubectl edit pv/<replace PV name> -n vault
```
Update the access mode from `ReadWriteOnce` to `ReadWriteMany` and save the configuration:
```yaml
spec:
  accessModes:
  - ReadWriteMany
```

## Step 4: Enable Audit Logs from the CLI
Access the Vault pod and enable audit logs:
```bash
kubectl exec -it pod/vault-0 -n vault -- sh
vault login root
vault audit list
vault audit enable file file_path=/vault/logs/vault-audit.log log_format=json
vault audit list
```

## Step 5: Install Fluent Bit and Add Helm Repository
Add the Fluent Bit Helm repository and update it:
```bash
helm repo add fluent https://fluent.github.io/helm-charts
helm repo update
```

## Step 6: Create a ConfigMap for Fluent Bit
Apply the ConfigMap for Fluent Bit configuration. 

### Note: Before creating the ConfigMap, update the following TODOs in `fb-configmap.yaml`:
1. Replace the file path (`#TODO: REPLACE PATH`).
2. Replace the host (`#TODO: REPLACE HOST`).
3. Replace the URI if needed (`#TODO: REPLACE URI IF NEEDED`).
4. Replace the Bearer token (`#TODO: REPLACE Bearer token`).

```bash
kubectl apply -f fb-configmap.yaml -n vault
kubectl get configmap -n vault
kubectl apply -f fb-deployment.yaml -n vault
```

## Step 7: Generate Logs and Verify Fluent Bit Output
Generate some logs by logging into the Vault UI and creating secrets.

Verify the Fluent Bit pod logs:
```bash
kubectl logs pod/<pod name> -n vault
```
You should see the recent audit log content.

## Step 8: Verify Logs in Apica Ascent
1. Log in to Apica Ascent.
2. Navigate to **Logs & Insights**.
3. Look for the `vault-logs` namespace.
4. Click on the Vault app to view the logs.
