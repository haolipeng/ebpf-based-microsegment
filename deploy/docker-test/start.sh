#!/bin/bash
# Quick Start Script for Docker Test Environment
#
# Usage:
#   ./start.sh          # Start simple environment
#   ./start.sh full     # Start full environment (requires building)
#   ./start.sh stop     # Stop all containers
#   ./start.sh clean    # Stop and remove all data

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

show_topology() {
    cat << 'EOF'

╔══════════════════════════════════════════════════════════════════╗
║                    Network Topology Test Environment             ║
╠══════════════════════════════════════════════════════════════════╣
║                                                                  ║
║   ┌──────────────┐                                               ║
║   │   client-1   │──┐                                            ║
║   │ (172.31.0.100)│  │                                           ║
║   └──────────────┘  │    ┌──────────────┐    ┌──────────────┐   ║
║                     ├───>│ nginx-proxy  │───>│   httpbin    │   ║
║   ┌──────────────┐  │    │ (172.31.0.10)│    │ (172.31.0.20)│   ║
║   │   client-2   │──┤    │   :8880      │    │              │   ║
║   │ (172.31.0.101)│  │    └──────────────┘    └──────────────┘   ║
║   └──────┬───────┘  │                                            ║
║          │          │                                            ║
║   ┌──────────────┐  │    ┌──────────────┐    ┌──────────────┐   ║
║   │   client-3   │──┘    │    redis     │    │   postgres   │   ║
║   │ (172.31.0.102)│──────>│ (172.31.0.30)│<───│ (172.31.0.40)│   ║
║   └──────────────┘       │   :6379      │    │   :5432      │   ║
║                          └──────────────┘    └──────────────┘   ║
║                                                                  ║
╚══════════════════════════════════════════════════════════════════╝

EOF
}

start_simple() {
    print_status "Starting simple test environment (no build required)..."
    show_topology

    docker-compose -f docker-compose-simple.yml up -d

    print_success "Environment started!"
    echo ""
    print_status "Services:"
    echo "  - Nginx Proxy:  http://localhost:8880"
    echo "  - HTTPBin API:  http://localhost:8880/get"
    echo "  - Redis:        localhost:6379 (internal)"
    echo "  - PostgreSQL:   localhost:5432 (internal)"
    echo ""
    print_status "View logs:"
    echo "  docker-compose -f docker-compose-simple.yml logs -f"
    echo ""
    print_status "View container traffic:"
    echo "  docker stats"
    echo ""
    print_status "To stop:"
    echo "  ./start.sh stop"
}

start_full() {
    print_status "Starting full test environment (with custom services)..."
    print_warning "This requires building Docker images, may take a few minutes..."

    docker-compose -f docker-compose.yml build
    docker-compose -f docker-compose.yml up -d

    print_success "Full environment started!"
    echo ""
    print_status "Services:"
    echo "  - Nginx Gateway:   http://localhost:8888"
    echo "  - RabbitMQ UI:     http://localhost:15672 (testuser/testpass)"
    echo "  - API Service 1:   172.28.0.20:8080"
    echo "  - API Service 2:   172.28.0.21:8080"
    echo ""
    print_status "View logs:"
    echo "  docker-compose -f docker-compose.yml logs -f"
}

stop_all() {
    print_status "Stopping all containers..."

    if [ -f docker-compose-simple.yml ]; then
        docker-compose -f docker-compose-simple.yml down 2>/dev/null || true
    fi

    if [ -f docker-compose.yml ]; then
        docker-compose -f docker-compose.yml down 2>/dev/null || true
    fi

    print_success "All containers stopped"
}

clean_all() {
    print_warning "This will remove all containers and volumes!"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_status "Cleaning up..."

        docker-compose -f docker-compose-simple.yml down -v 2>/dev/null || true
        docker-compose -f docker-compose.yml down -v 2>/dev/null || true

        print_success "Cleanup complete"
    else
        print_status "Cleanup cancelled"
    fi
}

show_status() {
    print_status "Container Status:"
    echo ""
    docker ps --filter "network=docker-test_test-net" \
              --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || \
    docker ps --filter "name=client-" --filter "name=nginx-" --filter "name=redis" \
              --filter "name=postgres" --filter "name=httpbin" \
              --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
    echo ""
    print_status "Network Traffic (last 10 seconds):"
    docker stats --no-stream --format "table {{.Name}}\t{{.NetIO}}\t{{.BlockIO}}" 2>/dev/null | head -10
}

# Main
case "${1:-simple}" in
    simple)
        start_simple
        ;;
    full)
        start_full
        ;;
    stop)
        stop_all
        ;;
    clean)
        clean_all
        ;;
    status)
        show_status
        ;;
    *)
        echo "Usage: $0 {simple|full|stop|clean|status}"
        echo ""
        echo "  simple  - Start simple environment (default, no build)"
        echo "  full    - Start full environment (requires building)"
        echo "  stop    - Stop all containers"
        echo "  clean   - Stop and remove all data"
        echo "  status  - Show container status"
        exit 1
        ;;
esac
