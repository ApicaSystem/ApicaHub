Configuring Vault in dev mode and ingest the audit logs into Apica Ascent - AWS.

prerequesites:
EKS cluster with addons (efs_csi_driver,pod_identity,roles & policies)

To Create a EKS Cluster with required components & drivers please Execute below tf snippet.

tf help:

terraform initialazation:
tereaform init

terraform plan:
terraform plan -out=plan.tf 

note: review the plan before apply

terraform apply:
terraform apply plan.tf



Step 1: Create a Namespace for Vault

Create a namespace to isolate Vault resources.

kubectl create namespace vault
kubectl get namespace

Step 2: Create a persistent volume vlaim (PVC) with access mode ReadWriteMany (EFS)

The persistent volume claim (PVC) is required to stores logs data, and fluentBit reads the logs data from this pvc and ingest it to Ascent.

Create the new PVC under vault namespace.

kubectl apply -f vault-efs-pvc.yaml -n vault
kubectl get pvc -n vault

Step 3: Install Vault in Dev Mode and Verify the Installation
The following commands installs the hashicorp vault in development mode.

helm repo add hashicorp https://helm.releases.hashicorp.com
helm repo update

helm upgrade --install vault hashicorp/vault \
       --set='server.dev.enabled=true' \
       --set='ui.enabled=true' \
       --set='ui.serviceType=LoadBalancer' \
       --namespace vault -f server.yaml --debug

kubectl get all -n vault

Verify the pods are in the running state, update the PVC access mode.

Step 4: Enable Audit Logs from the CLI

bydefault This below cmds will apply inside the vault pod:

kubectl exec -it pod/vault-0 -n vault -- sh
vault login root
vault audit list
vault audit enable file file_path=/vault/logs/vault-audit.log log_format=json
vault audit list

Step 5: Install Fluent Bit and Add Helm Repository

Add the Fluent Bit Helm repository and update it:

helm repo add fluent https://fluent.github.io/helm-charts
helm repo update

Step 6: Create a ConfigMap for Fluent Bit

Apply the ConfigMap for Fluent Bit configuration.

Note: Before creating the ConfigMap, update the following TODOs in fb-configmap.yaml:
Replace the file path (#TODO: REPLACE PATH).

Replace the host (#TODO: REPLACE HOST).

Replace the URI if needed (#TODO: REPLACE URI IF NEEDED).

Replace the Bearer token (#TODO: REPLACE Bearer token).

kubectl apply -f fb-configmap.yaml -n vault
kubectl get configmap -n vault
kubectl apply -f fb-deployment.yaml -n vault

Step 7: Generate Logs and Verify Fluent Bit Output

Generate some logs by logging into the Vault UI and creating secrets.

Verify the Fluent Bit pod logs:

kubectl logs pod/<pod name> -n vault

You should see the recent audit log content.

Step 8: Verify Logs in Apica Ascent
Log in to Apica Ascent.

Navigate to Logs & Insights.

Look for the vault-logs namespace.

Click on the Vault app to view the logs.