#!/bin/bash
set -euo pipefail

# =============================================================================
# k0s Cluster Setup Script for Splunk AI Platform
# =============================================================================
# Mirrors eks_cluster_with_stack.sh functionality but for k0s clusters
# Supports:
#   1. On-prem/baremetal: Use customer-provided IP addresses
#   2. AWS EC2: Automatically create EC2 instances for testing
# =============================================================================

# --- Unset conflicting AWS credentials ---
unset AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN AWS_PROFILE 2>/dev/null || true

# --- Non-interactive setup ---
export AWS_PAGER=""
export AWS_DEFAULT_OUTPUT=json
export PAGER=cat
export GIT_PAGER=cat
export LESS=FRX
export EDITOR=cat
export KUBE_EDITOR=cat
export LANG=C LC_ALL=C

# ====== CONFIG FILE LOCATION ======
CONFIG_FILE="${CONFIG_FILE:-$(dirname "$0")/k0s-cluster-config.yaml}"

# ====== COLORS & LOGGING ======
log()   { echo -e "\033[1;36m[INFO]\033[0m $*" >&2; }
warn()  { echo -e "\033[1;33m[WARN]\033[0m $*" >&2; }
err()   { echo -e "\033[1;31m[ERROR]\033[0m $*" >&2; exit 1; }
need()  { command -v "$1" >/dev/null 2>&1 || err "Missing $1 in PATH"; }

# ====== PREFLIGHT CHECKS ======
PF_FAILS=0; PF_WARN=0
pf_header(){ echo -e "\n\033[1;34m[CHECK]\033[0m $*" >&2; }
pf_ok()   { echo -e "  \033[1;32m✔\033[0m $*" >&2; }
pf_warn() { echo -e "  \033[1;33m!\033[0m $*" >&2; PF_WARN=$((PF_WARN+1)); }
pf_fail() { echo -e "  \033[1;31m✖\033[0m $*" >&2; PF_FAILS=$((PF_FAILS+1)); }
pf_summary(){
  echo -e "\n\033[1;34m[SUMMARY]\033[0m Preflight complete: \033[1;32m${PF_FAILS} error(s)\033[0m, \033[1;33m${PF_WARN} warning(s)\033[0m." >&2
  (( PF_FAILS == 0 )) || err "Preflight failed; please fix the above and rerun."
}

# ====== TEMP FILES ======
TMP_FILES=()
cleanup_tmp() { [[ ${#TMP_FILES[@]} -gt 0 ]] && rm -f "${TMP_FILES[@]}" 2>/dev/null || true; }
trap cleanup_tmp EXIT

# ====== LOAD CONFIGURATION ======
load_config() {
  log "Loading configuration from: ${CONFIG_FILE}"
  [[ -f "${CONFIG_FILE}" ]] || err "Config file not found: ${CONFIG_FILE}"

  # Parse YAML configuration
  CLUSTER_NAME=$(yq eval '.cluster.name' "${CONFIG_FILE}" 2>/dev/null || grep '^  name:' "${CONFIG_FILE}" | awk '{print $2}')
  REGION=$(yq eval '.cluster.region' "${CONFIG_FILE}" 2>/dev/null || grep '^  region:' "${CONFIG_FILE}" | awk '{print $2}')

  # Node IPs (for existing infrastructure)
  EXISTING_CONTROLLER_IPS=$(yq eval '.nodes.existingIPs.controllers[]' "${CONFIG_FILE}" 2>/dev/null | tr '\n' ' ' || echo "")
  EXISTING_WORKER_IPS=$(yq eval '.nodes.existingIPs.workers[]' "${CONFIG_FILE}" 2>/dev/null | tr '\n' ' ' || echo "")
  SSH_USER=$(yq eval '.cluster.sshUser' "${CONFIG_FILE}" 2>/dev/null || echo "ubuntu")
  SSH_KEY_PATH=$(yq eval '.cluster.sshKeyPath' "${CONFIG_FILE}" 2>/dev/null || echo "")

  # EC2 configuration (if creating instances)
  VPC_ID=$(yq eval '.ec2.vpcId' "${CONFIG_FILE}" 2>/dev/null || echo "")
  SUBNET_ID=$(yq eval '.ec2.subnetId' "${CONFIG_FILE}" 2>/dev/null || echo "")
  KEY_NAME=$(yq eval '.ec2.keyName' "${CONFIG_FILE}" 2>/dev/null || echo "")

  CONTROLLER_COUNT=$(yq eval '.nodes.controllers' "${CONFIG_FILE}" 2>/dev/null || echo "1")
  CPU_WORKER_COUNT=$(yq eval '.nodes.cpuWorkers' "${CONFIG_FILE}" 2>/dev/null || echo "2")
  GPU_WORKER_COUNT=$(yq eval '.nodes.gpuWorkers' "${CONFIG_FILE}" 2>/dev/null || echo "1")

  CONTROLLER_INSTANCE_TYPE=$(yq eval '.instanceTypes.controller' "${CONFIG_FILE}" 2>/dev/null || echo "t3.xlarge")
  CPU_WORKER_INSTANCE_TYPE=$(yq eval '.instanceTypes.cpuWorker' "${CONFIG_FILE}" 2>/dev/null || echo "m5.4xlarge")
  GPU_WORKER_INSTANCE_TYPE=$(yq eval '.instanceTypes.gpuWorker' "${CONFIG_FILE}" 2>/dev/null || echo "g5.2xlarge")

  # MinIO configuration
  MINIO_ACCESS_KEY=$(yq eval '.minio.accessKey' "${CONFIG_FILE}" 2>/dev/null || echo "minioadmin")
  MINIO_SECRET_KEY=$(yq eval '.minio.secretKey' "${CONFIG_FILE}" 2>/dev/null || echo "minioadmin123")
  MINIO_BUCKET=$(yq eval '.minio.bucket' "${CONFIG_FILE}" 2>/dev/null || echo "ai-platform-data")

  # Kubernetes namespace
  AI_NS=$(yq eval '.kubernetes.namespace' "${CONFIG_FILE}" 2>/dev/null || echo "ai-platform")

  # Splunk configuration
  AI_STANDALONE_NAME=$(yq eval '.splunk.standaloneName' "${CONFIG_FILE}" 2>/dev/null || echo "splunk-standalone")

  # Get AWS account if using EC2
  if [[ -z "${EXISTING_CONTROLLER_IPS}" ]]; then
    ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text 2>/dev/null || echo "")
  fi

  # File paths
  SPLUNK_OPERATOR_FILE=$(yq eval '.files.splunkOperator' "${CONFIG_FILE}" 2>/dev/null || echo "./splunk-operator-cluster.yaml")
  SPLUNK_AI_FILE=$(yq eval '.files.aiPlatform' "${CONFIG_FILE}" 2>/dev/null || echo "./artifacts.yaml")

  log "Configuration loaded: cluster=${CLUSTER_NAME}, namespace=${AI_NS}"
}

# ====== PREFLIGHT CHECKS ======
preflight_checks() {
  pf_header "Required tools"
  for tool in ssh kubectl helm git jq; do
    if command -v "$tool" >/dev/null 2>&1; then
      pf_ok "$tool found"
    else
      pf_fail "$tool not found in PATH"
    fi
  done

  # Check for yq
  if command -v yq >/dev/null 2>&1; then
    pf_ok "yq found"
  else
    pf_warn "yq not found - using fallback parsing (install yq for better results)"
  fi

  pf_header "Configuration"
  [[ -n "${CLUSTER_NAME}" ]] && pf_ok "Cluster name: ${CLUSTER_NAME}" || pf_fail "Cluster name not set"
  [[ -f "${SPLUNK_OPERATOR_FILE}" ]] && pf_ok "Splunk operator file: ${SPLUNK_OPERATOR_FILE}" || pf_warn "Splunk operator file not found: ${SPLUNK_OPERATOR_FILE}"
  [[ -f "${SPLUNK_AI_FILE}" ]] && pf_ok "AI platform file: ${SPLUNK_AI_FILE}" || pf_warn "AI platform file not found: ${SPLUNK_AI_FILE}"

  pf_header "Infrastructure mode"
  if [[ -n "${EXISTING_CONTROLLER_IPS}" ]]; then
    pf_ok "Using existing infrastructure (on-prem/baremetal)"
    pf_ok "Controller IPs: ${EXISTING_CONTROLLER_IPS}"
    pf_ok "Worker IPs: ${EXISTING_WORKER_IPS}"
    [[ -n "${SSH_KEY_PATH}" && -f "${SSH_KEY_PATH}" ]] && pf_ok "SSH key: ${SSH_KEY_PATH}" || pf_fail "SSH key not found: ${SSH_KEY_PATH}"
  else
    pf_ok "Creating EC2 instances"
    if command -v aws >/dev/null 2>&1; then
      pf_ok "AWS CLI found"
      [[ -n "${ACCOUNT_ID}" ]] && pf_ok "AWS Account: ${ACCOUNT_ID}" || pf_fail "Cannot get AWS account ID"
      [[ -n "${VPC_ID}" ]] && pf_ok "VPC ID: ${VPC_ID}" || pf_fail "VPC ID not set"
      [[ -n "${KEY_NAME}" ]] && pf_ok "EC2 Key name: ${KEY_NAME}" || pf_fail "EC2 key name not set"
    else
      pf_fail "AWS CLI not found - required for EC2 instance creation"
    fi
  fi

  pf_summary
}

# ====== SSH HELPER ======
ssh_exec() {
  local host="$1"
  shift
  local cmd="$*"

  if [[ -n "${SSH_KEY_PATH}" ]]; then
    ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i "${SSH_KEY_PATH}" "${SSH_USER}@${host}" "${cmd}"
  else
    ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "${SSH_USER}@${host}" "${cmd}"
  fi
}

scp_file() {
  local file="$1"
  local host="$2"
  local dest="$3"

  if [[ -n "${SSH_KEY_PATH}" ]]; then
    scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i "${SSH_KEY_PATH}" "${file}" "${SSH_USER}@${host}:${dest}"
  else
    scp -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null "${file}" "${SSH_USER}@${host}:${dest}"
  fi
}

# ====== EC2 INSTANCE CREATION ======
create_security_group() {
  log "Creating security group for k0s cluster..."

  local sg_name="${CLUSTER_NAME}-k0s-sg"
  local sg_id

  sg_id=$(aws ec2 describe-security-groups \
    --region "${REGION}" \
    --filters "Name=group-name,Values=${sg_name}" "Name=vpc-id,Values=${VPC_ID}" \
    --query 'SecurityGroups[0].GroupId' --output text 2>/dev/null || echo "None")

  if [[ "${sg_id}" != "None" && -n "${sg_id}" ]]; then
    log "Security group already exists: ${sg_id}"
    echo "${sg_id}"
    return 0
  fi

  sg_id=$(aws ec2 create-security-group \
    --region "${REGION}" \
    --group-name "${sg_name}" \
    --description "Security group for ${CLUSTER_NAME} k0s cluster" \
    --vpc-id "${VPC_ID}" \
    --query 'GroupId' --output text)

  log "Created security group: ${sg_id}"

  # Add ingress rules
  aws ec2 authorize-security-group-ingress --region "${REGION}" --group-id "${sg_id}" \
    --protocol tcp --port 6443 --source-group "${sg_id}" 2>/dev/null || true
  aws ec2 authorize-security-group-ingress --region "${REGION}" --group-id "${sg_id}" \
    --protocol tcp --port 2380 --source-group "${sg_id}" 2>/dev/null || true
  aws ec2 authorize-security-group-ingress --region "${REGION}" --group-id "${sg_id}" \
    --protocol tcp --port 10250 --source-group "${sg_id}" 2>/dev/null || true
  aws ec2 authorize-security-group-ingress --region "${REGION}" --group-id "${sg_id}" \
    --protocol tcp --port 30000-32767 --source-group "${sg_id}" 2>/dev/null || true
  aws ec2 authorize-security-group-ingress --region "${REGION}" --group-id "${sg_id}" \
    --protocol tcp --port 22 --cidr 0.0.0.0/0 2>/dev/null || true
  aws ec2 authorize-security-group-ingress --region "${REGION}" --group-id "${sg_id}" \
    --protocol -1 --source-group "${sg_id}" 2>/dev/null || true

  log "Security group rules configured"
  echo "${sg_id}"
}

create_ec2_instances() {
  log "Creating EC2 instances for k0s cluster..."

  local sg_id
  sg_id=$(create_security_group)

  # Get subnet if not provided
  if [[ -z "${SUBNET_ID}" ]]; then
    SUBNET_ID=$(aws ec2 describe-subnets \
      --region "${REGION}" \
      --filters "Name=vpc-id,Values=${VPC_ID}" \
      --query 'Subnets[0].SubnetId' --output text)
  fi

  [[ -n "${SUBNET_ID}" && "${SUBNET_ID}" != "None" ]] || err "No subnets found in VPC ${VPC_ID}"

  # Get latest Ubuntu 22.04 AMI
  local ami_id
  ami_id=$(aws ec2 describe-images \
    --region "${REGION}" \
    --owners 099720109477 \
    --filters "Name=name,Values=ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*" \
    --query 'sort_by(Images, &CreationDate)[-1].ImageId' --output text)

  log "Using AMI: ${ami_id}"

  # User data for k0s installation - write to temp file
  local user_data_file="/tmp/k0s-userdata-$$.sh"
  cat > "${user_data_file}" <<'EOF'
#!/bin/bash
set -ex
apt-get update
apt-get install -y curl wget jq
curl -sSLf https://get.k0s.sh | sh
EOF
  TMP_FILES+=("${user_data_file}")

  # Create instances (arrays already declared globally at top of script)
  CONTROLLER_IPS=()
  WORKER_IPS=()
  ALL_INSTANCE_IDS=()

  # Controllers
  log "Creating ${CONTROLLER_COUNT} controller(s)..."
  for ((i=0; i<CONTROLLER_COUNT; i++)); do
    local instance_id
    instance_id=$(aws ec2 run-instances \
      --region "${REGION}" \
      --image-id "${ami_id}" \
      --instance-type "${CONTROLLER_INSTANCE_TYPE}" \
      --key-name "${KEY_NAME}" \
      --security-group-ids "${sg_id}" \
      --subnet-id "${SUBNET_ID}" \
      --associate-public-ip-address \
      --user-data "file://${user_data_file}" \
      --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=${CLUSTER_NAME}-controller-${i}},{Key=Cluster,Value=${CLUSTER_NAME}},{Key=Role,Value=controller}]" \
      --block-device-mappings '[{"DeviceName":"/dev/sda1","Ebs":{"VolumeSize":100,"VolumeType":"gp3"}}]' \
      --query 'Instances[0].InstanceId' \
      --output text)

    ALL_INSTANCE_IDS+=("${instance_id}")
    log "Created controller: ${instance_id}"
  done

  # CPU Workers
  log "Creating ${CPU_WORKER_COUNT} CPU worker(s)..."
  for ((i=0; i<CPU_WORKER_COUNT; i++)); do
    local instance_id
    instance_id=$(aws ec2 run-instances \
      --region "${REGION}" \
      --image-id "${ami_id}" \
      --instance-type "${CPU_WORKER_INSTANCE_TYPE}" \
      --key-name "${KEY_NAME}" \
      --security-group-ids "${sg_id}" \
      --subnet-id "${SUBNET_ID}" \
      --associate-public-ip-address \
      --user-data "file://${user_data_file}" \
      --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=${CLUSTER_NAME}-cpu-worker-${i}},{Key=Cluster,Value=${CLUSTER_NAME}},{Key=Role,Value=cpu-worker}]" \
      --block-device-mappings '[{"DeviceName":"/dev/sda1","Ebs":{"VolumeSize":200,"VolumeType":"gp3"}}]' \
      --query 'Instances[0].InstanceId' \
      --output text)

    ALL_INSTANCE_IDS+=("${instance_id}")
    log "Created CPU worker: ${instance_id}"
  done

  # GPU Workers
  if [[ ${GPU_WORKER_COUNT} -gt 0 ]]; then
    log "Creating ${GPU_WORKER_COUNT} GPU worker(s)..."
    for ((i=0; i<GPU_WORKER_COUNT; i++)); do
      local instance_id
      instance_id=$(aws ec2 run-instances \
        --region "${REGION}" \
        --image-id "${ami_id}" \
        --instance-type "${GPU_WORKER_INSTANCE_TYPE}" \
        --key-name "${KEY_NAME}" \
        --security-group-ids "${sg_id}" \
        --subnet-id "${SUBNET_ID}" \
        --associate-public-ip-address \
        --user-data "file://${user_data_file}" \
        --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=${CLUSTER_NAME}-gpu-worker-${i}},{Key=Cluster,Value=${CLUSTER_NAME}},{Key=Role,Value=gpu-worker}]" \
        --block-device-mappings '[{"DeviceName":"/dev/sda1","Ebs":{"VolumeSize":300,"VolumeType":"gp3"}}]' \
        --query 'Instances[0].InstanceId' \
        --output text)

      ALL_INSTANCE_IDS+=("${instance_id}")
      log "Created GPU worker: ${instance_id}"
    done
  fi

  log "Waiting for instances to be running..."
  aws ec2 wait instance-running --region "${REGION}" --instance-ids "${ALL_INSTANCE_IDS[@]}"

  log "Waiting for instance status checks (this may take 3-5 minutes)..."
  aws ec2 wait instance-status-ok --region "${REGION}" --instance-ids "${ALL_INSTANCE_IDS[@]}" || true

  log "Waiting additional time for SSH to be fully ready..."
  sleep 60

  # Get IPs - use public IPs for SSH access from outside VPC
  for id in "${ALL_INSTANCE_IDS[@]}"; do
    local role
    role=$(aws ec2 describe-instances --region "${REGION}" --instance-ids "${id}" \
      --query 'Reservations[0].Instances[0].Tags[?Key==`Role`].Value' --output text)

    # Get public IP for SSH access
    local ip
    ip=$(aws ec2 describe-instances --region "${REGION}" --instance-ids "${id}" \
      --query 'Reservations[0].Instances[0].PublicIpAddress' --output text)

    # Fallback to private IP if no public IP (for VPC-internal access)
    if [[ -z "${ip}" || "${ip}" == "None" ]]; then
      ip=$(aws ec2 describe-instances --region "${REGION}" --instance-ids "${id}" \
        --query 'Reservations[0].Instances[0].PrivateIpAddress' --output text)
      log "Warning: Instance ${id} has no public IP, using private IP ${ip}"
    fi

    if [[ "${role}" == "controller" ]]; then
      CONTROLLER_IPS+=("${ip}")
      log "Controller IP: ${ip}"
    else
      WORKER_IPS+=("${ip}")
      log "Worker IP: ${ip} (${role})"
    fi
  done

  # Set SSH key path from EC2 key
  SSH_KEY_PATH="${HOME}/.ssh/${KEY_NAME}.pem"
}

# ====== K0S CLUSTER INSTALLATION ======
install_k0s_cluster() {
  log "Installing k0s cluster..."

  # Parse existing IPs if provided
  if [[ -n "${EXISTING_CONTROLLER_IPS}" ]]; then
    IFS=' ' read -ra CONTROLLER_IPS <<< "${EXISTING_CONTROLLER_IPS}"
    IFS=' ' read -ra WORKER_IPS <<< "${EXISTING_WORKER_IPS}"
  fi

  local controller_ip="${CONTROLLER_IPS[0]}"
  log "Primary controller: ${controller_ip}"

  # Generate k0s config
  log "Generating k0s configuration..."
  ssh_exec "${controller_ip}" "k0s config create > /tmp/k0s.yaml"

  # Customize config for AI workloads - merge patch with base config
  log "Applying configuration customizations for AI workloads..."
  ssh_exec "${controller_ip}" "cat <<'PATCH_EOF' > /tmp/k0s-config-patch.yaml
spec:
  storage:
    type: kine
  network:
    provider: calico
    calico:
      mode: vxlan
  extensions:
    helm:
      repositories:
      - name: stable
        url: https://charts.helm.sh/stable
PATCH_EOF"

  # Merge patch with base config using yq if available, otherwise use the base config
  ssh_exec "${controller_ip}" "if command -v yq >/dev/null 2>&1; then yq eval-all 'select(fileIndex == 0) * select(fileIndex == 1)' /tmp/k0s.yaml /tmp/k0s-config-patch.yaml > /tmp/k0s-merged.yaml && mv /tmp/k0s-merged.yaml /tmp/k0s.yaml; else echo 'yq not found, using base config'; fi"

  # Install k0s controller
  log "Installing k0s controller on ${controller_ip}..."
  ssh_exec "${controller_ip}" "sudo k0s install controller --config /tmp/k0s.yaml --enable-worker"
  ssh_exec "${controller_ip}" "sudo k0s start"

  log "Waiting for controller to be ready (60s)..."
  sleep 60

  # Generate worker token
  log "Generating worker join token..."
  local worker_token
  worker_token=$(ssh_exec "${controller_ip}" "sudo k0s token create --role=worker")

  # Install workers
  for worker_ip in "${WORKER_IPS[@]}"; do
    log "Installing k0s worker on ${worker_ip}..."
    ssh_exec "${worker_ip}" "echo '${worker_token}' | sudo k0s install worker --token-file=-" &
  done
  wait

  for worker_ip in "${WORKER_IPS[@]}"; do
    ssh_exec "${worker_ip}" "sudo k0s start" &
  done
  wait

  log "Waiting for workers to join (60s)..."
  sleep 60

  # Get kubeconfig
  log "Retrieving kubeconfig..."
  mkdir -p "${HOME}/.kube"
  ssh_exec "${controller_ip}" "sudo cat /var/lib/k0s/pki/admin.conf" > "${HOME}/.kube/k0s-${CLUSTER_NAME}"

  # Update server address
  sed -i.bak "s|server: .*|server: https://${controller_ip}:6443|" "${HOME}/.kube/k0s-${CLUSTER_NAME}"

  export KUBECONFIG="${HOME}/.kube/k0s-${CLUSTER_NAME}"

  log "k0s cluster installed successfully!"
  kubectl get nodes

  # Label nodes for proper workload scheduling
  label_nodes
}

# ====== LABEL NODES FOR WORKLOAD SCHEDULING ======
label_nodes() {
  log "Labeling nodes for AI workload scheduling..."

  # Wait for all nodes to be ready
  local node_count=$((${#CONTROLLER_IPS[@]} + ${#WORKER_IPS[@]}))
  log "Waiting for ${node_count} nodes to be ready..."

  local timeout=300
  local elapsed=0
  while [[ $(kubectl get nodes --no-headers | grep -c "Ready") -lt ${node_count} ]]; do
    sleep 5
    elapsed=$((elapsed + 5))
    if [[ ${elapsed} -ge ${timeout} ]]; then
      warn "Timeout waiting for all nodes to be ready, proceeding anyway..."
      break
    fi
  done

  # Get all nodes
  local all_nodes
  all_nodes=$(kubectl get nodes -o jsonpath='{.items[*].metadata.name}')

  # Label controller nodes
  for controller_ip in "${CONTROLLER_IPS[@]}"; do
    # Find node by IP
    local node_name
    node_name=$(kubectl get nodes -o json | jq -r ".items[] | select(.status.addresses[]? | select(.type==\"InternalIP\" and .address==\"${controller_ip}\")) | .metadata.name" | head -1)

    if [[ -n "${node_name}" ]]; then
      log "Labeling controller node: ${node_name}"
      kubectl label nodes "${node_name}" \
        splunk.ai/node-role=controller \
        splunk.ai/workload-type=control-plane \
        node.kubernetes.io/role=controller \
        --overwrite
    fi
  done

  # Label worker nodes based on their configuration
  local worker_index=0
  for worker_ip in "${WORKER_IPS[@]}"; do
    # Find node by IP
    local node_name
    node_name=$(kubectl get nodes -o json | jq -r ".items[] | select(.status.addresses[]? | select(.type==\"InternalIP\" and .address==\"${worker_ip}\")) | .metadata.name" | head -1)

    if [[ -n "${node_name}" ]]; then
      # Determine if this is a GPU or CPU worker based on index
      # First CPU_WORKER_COUNT workers are CPU, rest are GPU
      if [[ ${worker_index} -lt ${CPU_WORKER_COUNT} ]]; then
        log "Labeling CPU worker node: ${node_name}"
        kubectl label nodes "${node_name}" \
          splunk.ai/node-role=worker \
          splunk.ai/workload-type=cpu \
          node.kubernetes.io/workload=ai-cpu \
          splunk.ai/instance-type=cpu-worker \
          --overwrite
      else
        log "Labeling GPU worker node: ${node_name}"
        kubectl label nodes "${node_name}" \
          splunk.ai/node-role=worker \
          splunk.ai/workload-type=gpu \
          node.kubernetes.io/workload=ai-gpu \
          splunk.ai/instance-type=gpu-worker \
          nvidia.com/gpu=true \
          --overwrite
      fi
      worker_index=$((worker_index + 1))
    fi
  done

  # Add taints to GPU nodes to prevent non-GPU workloads from scheduling there
  log "Adding taints to GPU nodes..."
  kubectl get nodes -l splunk.ai/workload-type=gpu -o name | while read -r node; do
    kubectl taint nodes "${node#node/}" nvidia.com/gpu=true:NoSchedule --overwrite || true
  done

  log "Node labeling complete!"
  log "Nodes with labels:"
  kubectl get nodes --show-labels
}

# ====== WAIT FOR CRD ======
wait_for_crd() {
  local crd_name="$1"
  local timeout="${2:-300}"
  log "Waiting for CRD ${crd_name} (timeout: ${timeout}s)..."

  local elapsed=0
  while ! kubectl get crd "${crd_name}" >/dev/null 2>&1; do
    sleep 5
    elapsed=$((elapsed + 5))
    if [[ ${elapsed} -ge ${timeout} ]]; then
      err "Timeout waiting for CRD ${crd_name}"
    fi
  done
  log "CRD ${crd_name} is ready"
}

# ====== ENSURE NAMESPACE ======
ensure_namespace() {
  local ns="$1"
  if ! kubectl get namespace "${ns}" >/dev/null 2>&1; then
    log "Creating namespace ${ns}..."
    kubectl create namespace "${ns}"
  fi
}

# ====== INSTALL MINIO ======
install_minio() {
  log "Installing MinIO..."

  ensure_namespace "minio-system"

  # Create MinIO secret
  kubectl create secret generic minio-creds \
    --namespace=minio-system \
    --from-literal=accesskey="${MINIO_ACCESS_KEY}" \
    --from-literal=secretkey="${MINIO_SECRET_KEY}" \
    --dry-run=client -o yaml | kubectl apply -f -

  # Deploy MinIO
  cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: minio-pvc
  namespace: minio-system
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 200Gi
---
apiVersion: v1
kind: Service
metadata:
  name: minio
  namespace: minio-system
spec:
  type: ClusterIP
  ports:
    - port: 9000
      targetPort: 9000
      name: api
    - port: 9001
      targetPort: 9001
      name: console
  selector:
    app: minio
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: minio
  namespace: minio-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: minio
  template:
    metadata:
      labels:
        app: minio
    spec:
      containers:
      - name: minio
        image: minio/minio:latest
        args:
        - server
        - /data
        - --console-address
        - ":9001"
        env:
        - name: MINIO_ROOT_USER
          valueFrom:
            secretKeyRef:
              name: minio-creds
              key: accesskey
        - name: MINIO_ROOT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: minio-creds
              key: secretkey
        ports:
        - containerPort: 9000
          name: api
        - containerPort: 9001
          name: console
        volumeMounts:
        - name: data
          mountPath: /data
        resources:
          requests:
            cpu: "500m"
            memory: "2Gi"
          limits:
            cpu: "2"
            memory: "4Gi"
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: minio-pvc
EOF

  log "Waiting for MinIO to be ready..."
  kubectl wait --for=condition=ready pod -l app=minio -n minio-system --timeout=300s

  # Create bucket and directories using a job
  log "Creating MinIO bucket: ${MINIO_BUCKET}..."
  cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: minio-create-bucket
  namespace: minio-system
spec:
  template:
    spec:
      restartPolicy: OnFailure
      containers:
      - name: mc
        image: minio/mc:latest
        command:
        - /bin/sh
        - -c
        - |
          mc alias set myminio http://minio.minio-system.svc.cluster.local:9000 ${MINIO_ACCESS_KEY} ${MINIO_SECRET_KEY}
          mc mb myminio/${MINIO_BUCKET} || true
          mc anonymous set download myminio/${MINIO_BUCKET} || true

          # Create required directories in the bucket
          echo "Creating bucket directories..."
          echo "dummy" | mc pipe myminio/${MINIO_BUCKET}/apps/.keep
          echo "dummy" | mc pipe myminio/${MINIO_BUCKET}/tasks/.keep
          echo "dummy" | mc pipe myminio/${MINIO_BUCKET}/model_artifacts/.keep
          echo "dummy" | mc pipe myminio/${MINIO_BUCKET}/artifacts/.keep

          echo "Bucket and directories created successfully"
          mc ls myminio/${MINIO_BUCKET}/
EOF

  kubectl wait --for=condition=complete job/minio-create-bucket -n minio-system --timeout=120s || true

  log "MinIO bucket and directories created successfully"
}

# ====== INSTALL CERT-MANAGER ======
install_cert_manager() {
  log "Installing cert-manager..."

  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

  wait_for_crd certificates.cert-manager.io 300
  kubectl wait --for=condition=ready pod -l app.kubernetes.io/instance=cert-manager -n cert-manager --timeout=300s

  log "cert-manager installed successfully"
}

# ====== INSTALL NVIDIA GPU OPERATOR ======
install_nvidia_device_plugin() {
  if [[ ${GPU_WORKER_COUNT} -eq 0 ]]; then
    log "Skipping NVIDIA GPU operator (no GPU workers)"
    return 0
  fi

  log "Installing NVIDIA GPU Operator..."

  helm repo add nvidia https://helm.ngc.nvidia.com/nvidia || true
  helm repo update

  helm upgrade --install gpu-operator nvidia/gpu-operator \
    --namespace gpu-operator --create-namespace \
    --set driver.enabled=true \
    --set toolkit.enabled=true \
    --wait --timeout=10m

  log "NVIDIA GPU Operator installed successfully"
}

# ====== INSTALL PROMETHEUS OPERATOR ======
install_kube_prometheus() {
  log "Installing kube-prometheus-stack..."

  helm repo add prometheus-community https://prometheus-community.github.io/helm-charts || true
  helm repo update

  helm upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
    --namespace monitoring --create-namespace \
    --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
    --set prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues=false \
    --wait --timeout=10m

  log "kube-prometheus-stack installed successfully"
}

# ====== INSTALL OTEL OPERATOR ======
install_otel_operator_and_contrib_collector() {
  log "Installing OpenTelemetry Operator..."

  helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts || true
  helm repo update

  helm upgrade --install opentelemetry-operator open-telemetry/opentelemetry-operator \
    --namespace opentelemetry-operator-system --create-namespace \
    --set manager.collectorImage.repository=otel/opentelemetry-collector-contrib \
    --wait --timeout=10m

  wait_for_crd opentelemetrycollectors.opentelemetry.io 300

  log "OpenTelemetry Operator installed successfully"
}

# ====== INSTALL RAY OPERATOR ======
install_ray_operator() {
  log "Installing KubeRay Operator..."

  helm repo add kuberay https://ray-project.github.io/kuberay-helm/ || true
  helm repo update

  helm upgrade --install kuberay-operator kuberay/kuberay-operator \
    --namespace ray-system --create-namespace \
    --version 1.0.0 \
    --wait --timeout=10m

  wait_for_crd rayservices.ray.io 300
  wait_for_crd rayclusters.ray.io 300

  log "KubeRay Operator installed successfully"
}

# ====== INSTALL SPLUNK OPERATOR ======
install_splunk_operator() {
  log "Installing Splunk Operator..."

  if [[ ! -f "${SPLUNK_OPERATOR_FILE}" ]]; then
    warn "Splunk operator file not found: ${SPLUNK_OPERATOR_FILE}"
    return 0
  fi

  kubectl apply -f "${SPLUNK_OPERATOR_FILE}"

  wait_for_crd standalones.enterprise.splunk.com 300

  log "Splunk Operator installed successfully"
}

# ====== INSTALL SPLUNK AI OPERATOR ======
install_splunk_ai_operator() {
  log "Installing Splunk AI Platform Operator..."

  # Check if operator repo is cloned
  local operator_dir="/tmp/splunk-ai-operator"
  if [[ ! -d "${operator_dir}" ]]; then
    log "Cloning Splunk AI Operator repository..."
    git clone https://github.com/splunk/splunk-ai-operator.git "${operator_dir}"
  fi

  cd "${operator_dir}"

  # Install CRDs
  log "Installing AI Platform CRDs..."
  make install

  # Deploy operator
  log "Deploying AI Platform Operator..."
  make deploy IMG=splunk/splunk-ai-operator:latest

  # Wait for operator
  kubectl wait --for=condition=ready pod -l control-plane=controller-manager \
    -n splunk-ai-operator-system --timeout=600s

  wait_for_crd aiplatforms.ai.splunk.com 300
  wait_for_crd aiservices.ai.splunk.com 300

  log "Splunk AI Platform Operator installed successfully"
}

# ====== CREATE MINIO SECRET FOR AI PLATFORM ======
create_minio_secret() {
  local ns="$1"
  ensure_namespace "${ns}"

  log "Creating MinIO credentials secret in ${ns}..."

  kubectl create secret generic minio-credentials \
    --namespace="${ns}" \
    --from-literal=accessKey="${MINIO_ACCESS_KEY}" \
    --from-literal=secretKey="${MINIO_SECRET_KEY}" \
    --dry-run=client -o yaml | kubectl apply -f -

  log "MinIO credentials secret created"
  echo "minio-credentials"
}

# ====== INSTALL SPLUNK STANDALONE ======
install_splunk_standalone() {
  log "Installing Splunk Standalone: ${AI_STANDALONE_NAME} in ${AI_NS}..."

  ensure_namespace "${AI_NS}"
  wait_for_crd standalones.enterprise.splunk.com 600

  # Create MinIO secret for Splunk
  local minio_secret
  minio_secret=$(create_minio_secret "${AI_NS}")

  # Create Splunk Standalone with MinIO backend
  cat <<EOF | kubectl apply -f -
apiVersion: enterprise.splunk.com/v4
kind: Standalone
metadata:
  name: ${AI_STANDALONE_NAME}
  namespace: ${AI_NS}
spec:
  replicas: 1

  # Use MinIO for SmartStore
  volumes:
    - name: minio-vol
      persistentVolumeClaim:
        claimName: splunk-minio-pvc

  defaults: |-
    splunk:
      smartstore:
        - name: minio
          path: minio://${MINIO_BUCKET}/splunk-index
          remote.s3.endpoint: http://minio.minio-system.svc.cluster.local:9000
          remote.s3.access_key: ${MINIO_ACCESS_KEY}
          remote.s3.secret_key: ${MINIO_SECRET_KEY}
          remote.s3.signature_version: v4

      indexes:
        default:
          remotePath: volume:minio/$_index_name
EOF

  log "Waiting for Splunk Standalone to be ready..."
  kubectl wait --for=condition=ready pod -l app.kubernetes.io/instance=${AI_STANDALONE_NAME} -n ${AI_NS} --timeout=600s || true

  log "Splunk Standalone installed successfully"
}

# ====== INSTALL AI PLATFORM CR ======
install_ai_platform_cr() {
  local minio_secret="$1"

  log "Installing AIPlatform custom resource..."

  if [[ ! -f "${SPLUNK_AI_FILE}" ]]; then
    warn "AI Platform file not found: ${SPLUNK_AI_FILE}"
    log "Creating default AIPlatform CR..."

    cat <<EOF | kubectl apply -f -
apiVersion: ai.splunk.com/v1
kind: AIPlatform
metadata:
  name: ${CLUSTER_NAME}-ai-platform
  namespace: ${AI_NS}
spec:
  clusterDomain: cluster.local
  replicas: 1

  # MinIO object storage
  objectStorage:
    path: minio://${MINIO_BUCKET}/ai-platform
    endpoint: http://minio.minio-system.svc.cluster.local:9000
    region: us-east-1
    credentialsSecret: ${minio_secret}

  # Vector database
  storage:
    vectorDB:
      size: "50Gi"

  # CPU scheduling - use node labels to schedule on CPU workers
  cpuSchedulingSpec:
    nodeSelector:
      splunk.ai/workload-type: cpu
    affinity:
      nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          nodeSelectorTerms:
          - matchExpressions:
            - key: splunk.ai/workload-type
              operator: In
              values:
              - cpu
    tolerations: []

  # Worker configuration
  workerGroupConfig:
    imageRegistry: rayproject/ray:2.9.0
    minReplicas: 1
    maxReplicas: 5
    resourcesPerWorker:
      cpu: "4"
      memory: "16Gi"

  # GPU configuration - use node labels and tolerations for GPU nodes
  gpuSchedulingSpec:
    enabled: $([[ ${GPU_WORKER_COUNT} -gt 0 ]] && echo "true" || echo "false")
    nodeSelector:
      splunk.ai/workload-type: gpu
      nvidia.com/gpu: "true"
    affinity:
      nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          nodeSelectorTerms:
          - matchExpressions:
            - key: splunk.ai/workload-type
              operator: In
              values:
              - gpu
            - key: nvidia.com/gpu
              operator: In
              values:
              - "true"
    tolerations:
    - key: nvidia.com/gpu
      operator: Equal
      value: "true"
      effect: NoSchedule
    resourcesPerWorker:
      cpu: "8"
      memory: "32Gi"
      nvidia.com/gpu: "1"

  # Splunk configuration
  splunkConfiguration:
    endpoint: http://${AI_STANDALONE_NAME}-standalone-service.${AI_NS}.svc.cluster.local:8088
    secretRef:
      name: splunk-hec-secret
EOF
  else
    log "Applying AI Platform from file: ${SPLUNK_AI_FILE}"
    kubectl apply -f "${SPLUNK_AI_FILE}"
  fi

  log "AIPlatform CR installed successfully"
}

# ====== INSTALL FULL STACK ======
install_ai_platform_stack() {
  log "Installing complete AI Platform stack..."

  ensure_namespace "${AI_NS}"

  # Install infrastructure components
  install_minio
  install_cert_manager
  install_kube_prometheus
  install_otel_operator_and_contrib_collector
  install_nvidia_device_plugin
  install_ray_operator

  # Install Splunk components
  install_splunk_operator
  install_splunk_standalone

  # Install AI Platform operator
  install_splunk_ai_operator

  # Create MinIO secret and install AI Platform CR
  local minio_secret
  minio_secret=$(create_minio_secret "${AI_NS}")
  install_ai_platform_cr "${minio_secret}"

  log "AI Platform stack installation complete!"
}

# ====== MAIN INSTALL FLOW ======
main_install() {
  log "Starting k0s cluster installation with AI Platform stack..."

  load_config
  preflight_checks

  # Setup infrastructure
  if [[ -n "${EXISTING_CONTROLLER_IPS}" ]]; then
    log "Using existing infrastructure..."
  else
    log "Creating EC2 instances..."
    create_ec2_instances
  fi

  # Install k0s cluster
  install_k0s_cluster

  # Install AI Platform stack
  install_ai_platform_stack

  log "============================================"
  log "Installation complete!"
  log "============================================"
  log ""
  log "Kubeconfig: ${HOME}/.kube/k0s-${CLUSTER_NAME}"
  log "Set: export KUBECONFIG=${HOME}/.kube/k0s-${CLUSTER_NAME}"
  log ""
  log "MinIO Console: kubectl port-forward svc/minio -n minio-system 9001:9001"
  log "MinIO Credentials: ${MINIO_ACCESS_KEY} / ${MINIO_SECRET_KEY}"
  log ""
  log "Check cluster: kubectl get nodes"
  log "Check AI Platform: kubectl get aiplatform -n ${AI_NS}"
  log "Check Splunk: kubectl get standalone -n ${AI_NS}"
}

# ====== MAIN DELETE FLOW ======
main_delete() {
  load_config

  log "============================================"
  log "Starting cleanup of k0s cluster: ${CLUSTER_NAME}"
  log "============================================"

  # Set kubeconfig if cluster is still accessible
  export KUBECONFIG="${HOME}/.kube/k0s-${CLUSTER_NAME}"

  if [[ -f "${KUBECONFIG}" ]] && kubectl cluster-info &>/dev/null; then
    log "Cluster is accessible, performing graceful cleanup..."

    # Delete AI Platform resources
    log "Deleting AI Platform resources..."
    kubectl delete aiplatform --all -n "${AI_NS}" --timeout=120s || true
    kubectl delete aiservice --all -n "${AI_NS}" --timeout=120s || true

    # Delete Splunk resources
    log "Deleting Splunk Standalone..."
    kubectl delete standalone --all -n "${AI_NS}" --timeout=120s || true

    # Delete Ray resources
    log "Deleting Ray services..."
    kubectl delete rayservice --all -n "${AI_NS}" --timeout=120s || true
    kubectl delete raycluster --all -n "${AI_NS}" --timeout=120s || true

    # Delete namespace (this will cleanup remaining resources)
    log "Deleting namespace: ${AI_NS}..."
    kubectl delete namespace "${AI_NS}" --timeout=180s || true

    # Delete operators
    log "Deleting Splunk AI Operator..."
    kubectl delete namespace splunk-ai-operator-system --timeout=120s || true

    log "Deleting monitoring stack..."
    helm uninstall kube-prometheus-stack -n monitoring || true
    kubectl delete namespace monitoring --timeout=120s || true

    log "Deleting OpenTelemetry Operator..."
    helm uninstall opentelemetry-operator -n opentelemetry-operator-system || true
    kubectl delete namespace opentelemetry-operator-system --timeout=120s || true

    log "Deleting Ray Operator..."
    helm uninstall kuberay-operator -n ray-system || true
    kubectl delete namespace ray-system --timeout=120s || true

    log "Deleting GPU Operator..."
    helm uninstall gpu-operator -n gpu-operator --timeout=300s || true
    kubectl delete namespace gpu-operator --timeout=120s || true

    log "Deleting cert-manager..."
    kubectl delete -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml --timeout=120s || true

    log "Deleting MinIO..."
    kubectl delete namespace minio-system --timeout=120s || true

    log "Waiting for resource cleanup (30s)..."
    sleep 30
  else
    warn "Cluster not accessible or kubeconfig missing, skipping Kubernetes resource cleanup"
  fi

  # Stop and reset k0s on all nodes
  log "Stopping k0s on all nodes..."

  if [[ -n "${EXISTING_CONTROLLER_IPS}" ]]; then
    # On-prem: Stop k0s on existing infrastructure
    IFS=' ' read -ra CONTROLLER_IPS <<< "${EXISTING_CONTROLLER_IPS}"
    IFS=' ' read -ra WORKER_IPS <<< "${EXISTING_WORKER_IPS}"

    log "Stopping k0s on controller nodes..."
    for ip in "${CONTROLLER_IPS[@]}"; do
      log "  Stopping k0s on controller: ${ip}..."
      ssh_exec "${ip}" "sudo k0s stop || true; sudo k0s reset --force || true" || warn "Failed to stop k0s on ${ip}"
    done

    log "Stopping k0s on worker nodes..."
    for ip in "${WORKER_IPS[@]}"; do
      log "  Stopping k0s on worker: ${ip}..."
      ssh_exec "${ip}" "sudo k0s stop || true; sudo k0s reset --force || true" || warn "Failed to stop k0s on ${ip}"
    done

    log "k0s stopped on all on-prem nodes"
    log "NOTE: Node machines are still running. To clean up completely:"
    log "  - Remove k0s binaries: sudo rm -f /usr/local/bin/k0s"
    log "  - Clean up data: sudo rm -rf /var/lib/k0s /etc/k0s"

  else
    # EC2: Terminate instances
    log "Deleting EC2 instances..."

    local instance_ids
    instance_ids=$(aws ec2 describe-instances \
      --region "${REGION}" \
      --filters "Name=tag:Cluster,Values=${CLUSTER_NAME}" "Name=instance-state-name,Values=running,stopped,stopping" \
      --query 'Reservations[].Instances[].InstanceId' --output text)

    if [[ -n "${instance_ids}" ]]; then
      log "Found instances to terminate: ${instance_ids}"
      log "Terminating EC2 instances..."
      aws ec2 terminate-instances --region "${REGION}" --instance-ids ${instance_ids}

      log "Waiting for instances to terminate..."
      aws ec2 wait instance-terminated --region "${REGION}" --instance-ids ${instance_ids} || warn "Timeout waiting for instances to terminate"

      log "EC2 instances terminated"
    else
      log "No EC2 instances found with cluster tag"
    fi

    # Delete security group
    log "Deleting security group..."
    local sg_id
    sg_id=$(aws ec2 describe-security-groups \
      --region "${REGION}" \
      --filters "Name=group-name,Values=${CLUSTER_NAME}-k0s-sg" \
      --query 'SecurityGroups[0].GroupId' --output text 2>/dev/null || echo "")

    if [[ -n "${sg_id}" && "${sg_id}" != "None" ]]; then
      log "Deleting security group: ${sg_id}"
      # Wait a bit for ENIs to detach
      sleep 10
      aws ec2 delete-security-group --region "${REGION}" --group-id "${sg_id}" 2>/dev/null || warn "Could not delete security group (may have dependencies, will be auto-cleaned)"
    else
      log "Security group not found or already deleted"
    fi

    # Delete any EBS volumes that were created
    log "Checking for orphaned EBS volumes..."
    local volumes
    volumes=$(aws ec2 describe-volumes \
      --region "${REGION}" \
      --filters "Name=tag:Cluster,Values=${CLUSTER_NAME}" "Name=status,Values=available" \
      --query 'Volumes[].VolumeId' --output text)

    if [[ -n "${volumes}" ]]; then
      log "Deleting orphaned EBS volumes: ${volumes}"
      for vol in ${volumes}; do
        aws ec2 delete-volume --region "${REGION}" --volume-id "${vol}" || warn "Could not delete volume ${vol}"
      done
    fi
  fi

  # Clean up local files
  log "Cleaning up local files..."
  rm -f "${HOME}/.kube/k0s-${CLUSTER_NAME}" "${HOME}/.kube/k0s-${CLUSTER_NAME}.bak"
  rm -rf "/tmp/splunk-ai-operator" || true

  log "============================================"
  log "Cleanup complete!"
  log "============================================"
  log ""
  log "Cluster '${CLUSTER_NAME}' has been deleted."

  if [[ -n "${EXISTING_CONTROLLER_IPS}" ]]; then
    log ""
    log "On-prem nodes are still running with k0s stopped."
    log "To fully clean up each node, run:"
    log "  sudo rm -f /usr/local/bin/k0s"
    log "  sudo rm -rf /var/lib/k0s /etc/k0s"
  fi
}

# ====== CLEAN ALL (AGGRESSIVE CLEANUP) ======
clean_all() {
  log "============================================"
  log "AGGRESSIVE CLEANUP MODE"
  log "============================================"
  warn "This will forcefully remove all resources and data!"

  load_config

  # Run normal delete first
  main_delete

  # Additional aggressive cleanup for on-prem
  if [[ -n "${EXISTING_CONTROLLER_IPS}" ]]; then
    IFS=' ' read -ra CONTROLLER_IPS <<< "${EXISTING_CONTROLLER_IPS}"
    IFS=' ' read -ra WORKER_IPS <<< "${EXISTING_WORKER_IPS}"

    log "Performing aggressive cleanup on nodes..."
    for ip in "${CONTROLLER_IPS[@]}" "${WORKER_IPS[@]}"; do
      log "  Deep cleaning node: ${ip}..."
      ssh_exec "${ip}" "
        sudo systemctl stop k0scontroller k0sworker || true
        sudo systemctl disable k0scontroller k0sworker || true
        sudo rm -rf /var/lib/k0s /etc/k0s
        sudo rm -f /usr/local/bin/k0s
        sudo rm -rf /var/lib/kubelet /etc/cni /opt/cni
        sudo rm -rf /var/lib/calico /etc/calico
        sudo iptables -F || true
        sudo iptables -X || true
        sudo iptables -t nat -F || true
        sudo iptables -t nat -X || true
        sudo iptables -t mangle -F || true
        sudo iptables -t mangle -X || true
      " || warn "Failed aggressive cleanup on ${ip}"
    done
  fi

  log "Aggressive cleanup complete!"
}

# ====== USAGE ======
usage() {
  cat <<EOF
Usage: $0 [install|delete|clean-all]

Deploys Splunk AI Platform on k0s cluster (on-prem or EC2)

Commands:
  install    - Install k0s cluster and AI Platform stack
  delete     - Delete cluster and all resources (graceful)
  clean-all  - Aggressive cleanup including node-level cleanup (on-prem)

Environment:
  CONFIG_FILE - Path to k0s config YAML (default: ./k0s-cluster-config.yaml)

Examples:
  # On-prem with existing IPs
  CONFIG_FILE=./on-prem-config.yaml $0 install

  # EC2 simulation
  CONFIG_FILE=./ec2-config.yaml $0 install

  # Delete cluster (graceful)
  CONFIG_FILE=./config.yaml $0 delete

  # Deep cleanup (aggressive)
  CONFIG_FILE=./config.yaml $0 clean-all

Notes:
  - 'delete' performs graceful Kubernetes resource cleanup
  - 'clean-all' does aggressive node-level cleanup (removes all k0s files)
  - For EC2 mode, both commands terminate instances
  - For on-prem mode, machines remain running but k0s is removed
EOF
}

# ====== MAIN ======
case "${1:-install}" in
  install)
    main_install
    ;;
  delete)
    main_delete
    ;;
  clean-all)
    clean_all
    ;;
  *)
    usage
    exit 1
    ;;
esac
