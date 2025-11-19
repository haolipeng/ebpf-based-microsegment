#!/bin/bash

# Deploy microsegment to Kubernetes cluster
# This script deploys all components in the correct order

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

# Function to check cluster connection
check_cluster() {
    if ! kubectl cluster-info &> /dev/null; then
        error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi
    info "Connected to Kubernetes cluster"
}

# Function to wait for deployment to be ready
wait_for_deployment() {
    local deployment=$1
    local namespace=$2
    local timeout=${3:-300}

    info "Waiting for deployment ${deployment} to be ready (timeout: ${timeout}s)..."
    if kubectl wait --for=condition=available --timeout=${timeout}s \
        deployment/${deployment} -n ${namespace} &> /dev/null; then
        info "Deployment ${deployment} is ready"
        return 0
    else
        warn "Deployment ${deployment} is not ready after ${timeout}s"
        return 1
    fi
}

# Function to wait for daemonset to be ready
wait_for_daemonset() {
    local daemonset=$1
    local namespace=$2
    local timeout=${3:-300}

    info "Waiting for daemonset ${daemonset} to be ready (timeout: ${timeout}s)..."

    local end_time=$((SECONDS + timeout))
    while [ $SECONDS -lt $end_time ]; do
        local desired=$(kubectl get daemonset ${daemonset} -n ${namespace} -o jsonpath='{.status.desiredNumberScheduled}' 2>/dev/null || echo "0")
        local ready=$(kubectl get daemonset ${daemonset} -n ${namespace} -o jsonpath='{.status.numberReady}' 2>/dev/null || echo "0")

        if [ "$desired" != "0" ] && [ "$desired" == "$ready" ]; then
            info "DaemonSet ${daemonset} is ready (${ready}/${desired} pods)"
            return 0
        fi

        echo -n "."
        sleep 5
    done

    echo ""
    warn "DaemonSet ${daemonset} is not ready after ${timeout}s"
    return 1
}

# Main deployment function
deploy() {
    info "Starting Kubernetes deployment..."

    # Check prerequisites
    check_kubectl
    check_cluster

    # 1. Create namespace
    info "Creating namespace ${NAMESPACE}..."
    kubectl apply -f "${K8S_DIR}/namespace.yaml"

    # 2. Create RBAC resources
    info "Creating RBAC resources..."
    kubectl apply -f "${K8S_DIR}/rbac.yaml"

    # 3. Create ConfigMap
    info "Creating ConfigMap..."
    kubectl apply -f "${K8S_DIR}/configmap.yaml"

    # 4. Deploy PostgreSQL
    info "Deploying PostgreSQL..."
    kubectl apply -f "${K8S_DIR}/postgres.yaml"
    wait_for_deployment "microsegment-postgres" "${NAMESPACE}" 120

    # 5. Deploy Server
    info "Deploying Server..."
    kubectl apply -f "${K8S_DIR}/server.yaml"
    wait_for_deployment "microsegment-server" "${NAMESPACE}" 120

    # 6. Deploy Agent
    info "Deploying Agent DaemonSet..."
    kubectl apply -f "${K8S_DIR}/agent.yaml"
    wait_for_daemonset "microsegment-agent" "${NAMESPACE}" 120

    # Show deployment status
    info "Deployment completed!"
    echo ""
    info "Deployment status:"
    kubectl get all -n ${NAMESPACE}

    echo ""
    info "To check logs:"
    echo "  Server:    kubectl logs -f deployment/microsegment-server -n ${NAMESPACE}"
    echo "  Agent:     kubectl logs -f daemonset/microsegment-agent -n ${NAMESPACE}"
    echo "  Postgres:  kubectl logs -f deployment/microsegment-postgres -n ${NAMESPACE}"

    echo ""
    info "To access the server (port-forward):"
    echo "  kubectl port-forward -n ${NAMESPACE} svc/microsegment-server 8080:8080 9090:9090"
}

# Run deployment
deploy
