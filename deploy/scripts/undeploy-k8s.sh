#!/bin/bash

# Remove microsegment deployment from Kubernetes cluster
# This script removes all deployed resources

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
K8S_DIR="${PROJECT_ROOT}/deploy/kubernetes"

# Default namespace
NAMESPACE="microsegment"

# Function to print colored messages
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if kubectl is available
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        error "kubectl command not found. Please install kubectl first."
        exit 1
    fi
    info "kubectl is available"
}

# Function to check if namespace exists
check_namespace() {
    if ! kubectl get namespace ${NAMESPACE} &> /dev/null; then
        warn "Namespace ${NAMESPACE} does not exist. Nothing to undeploy."
        exit 0
    fi
}

# Main undeploy function
undeploy() {
    info "Starting Kubernetes undeployment..."

    # Check prerequisites
    check_kubectl
    check_namespace

    # Delete resources in reverse order

    # 1. Delete Agent DaemonSet
    info "Deleting Agent DaemonSet..."
    kubectl delete -f "${K8S_DIR}/agent.yaml" --ignore-not-found=true

    # 2. Delete Server
    info "Deleting Server..."
    kubectl delete -f "${K8S_DIR}/server.yaml" --ignore-not-found=true

    # 3. Delete PostgreSQL
    info "Deleting PostgreSQL..."
    kubectl delete -f "${K8S_DIR}/postgres.yaml" --ignore-not-found=true

    # 4. Delete ConfigMap
    info "Deleting ConfigMap..."
    kubectl delete -f "${K8S_DIR}/configmap.yaml" --ignore-not-found=true

    # 5. Delete RBAC resources
    info "Deleting RBAC resources..."
    kubectl delete -f "${K8S_DIR}/rbac.yaml" --ignore-not-found=true

    # 6. Delete namespace (optional - ask for confirmation)
    echo ""
    read -p "Do you want to delete the namespace '${NAMESPACE}'? (y/N): " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        info "Deleting namespace ${NAMESPACE}..."
        kubectl delete -f "${K8S_DIR}/namespace.yaml" --ignore-not-found=true
    else
        info "Keeping namespace ${NAMESPACE}"
    fi

    info "Undeployment completed!"
}

# Run undeployment
undeploy
