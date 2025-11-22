"""
Traffic Generator - Generates continuous network traffic for topology testing

This script generates various types of network traffic:
- HTTP requests to API services
- Redis cache operations
- Simulated burst traffic
- Random endpoint calls
"""

import os
import time
import random
import threading
import logging
import socket

import requests

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Configuration
TARGET_HOSTS = os.environ.get('TARGET_HOSTS', 'nginx-gateway,api-service-1,api-service-2').split(',')
INTERVAL = float(os.environ.get('INTERVAL', 2))
CONCURRENT = int(os.environ.get('CONCURRENT', 5))

# API endpoints to call
ENDPOINTS = [
    '/health',
    '/api/info',
    '/api/cache/get/test_key',
    '/api/cache/set/test_key/test_value',
    '/api/db/query',
    '/api/db/users',
    '/api/simulate/load',
]


def make_request(host, endpoint, port=8080):
    """Make HTTP request to a service"""
    url = f"http://{host}:{port}{endpoint}"
    try:
        start = time.time()
        resp = requests.get(url, timeout=10)
        duration = (time.time() - start) * 1000
        logger.info(f"[{resp.status_code}] {url} - {duration:.2f}ms")
        return resp.status_code == 200
    except requests.exceptions.ConnectionError:
        logger.warning(f"Connection failed: {url}")
        return False
    except requests.exceptions.Timeout:
        logger.warning(f"Timeout: {url}")
        return False
    except Exception as e:
        logger.error(f"Error calling {url}: {e}")
        return False


def post_request(host, endpoint, data, port=8080):
    """Make POST request"""
    url = f"http://{host}:{port}{endpoint}"
    try:
        start = time.time()
        resp = requests.post(url, json=data, timeout=10)
        duration = (time.time() - start) * 1000
        logger.info(f"[POST {resp.status_code}] {url} - {duration:.2f}ms")
        return resp.status_code == 200
    except Exception as e:
        logger.error(f"Error posting to {url}: {e}")
        return False


def redis_operations():
    """Generate Redis traffic"""
    try:
        import redis
        r = redis.Redis(host='redis-cache', port=6379, decode_responses=True)

        # Random operations
        key = f"traffic_test_{random.randint(1, 100)}"
        r.set(key, f"value_{time.time()}", ex=60)
        r.get(key)
        r.incr('traffic_counter')
        logger.info(f"Redis operations completed: {key}")
        return True
    except Exception as e:
        logger.error(f"Redis error: {e}")
        return False


def generate_burst_traffic(target, count=10):
    """Generate burst of traffic to a single target"""
    logger.info(f"Starting burst traffic to {target} ({count} requests)")
    threads = []
    for _ in range(count):
        endpoint = random.choice(ENDPOINTS)
        t = threading.Thread(target=make_request, args=(target, endpoint))
        threads.append(t)
        t.start()

    for t in threads:
        t.join()
    logger.info(f"Burst traffic to {target} completed")


def traffic_pattern_normal():
    """Normal traffic pattern - random calls"""
    host = random.choice(TARGET_HOSTS)
    endpoint = random.choice(ENDPOINTS)
    make_request(host, endpoint)


def traffic_pattern_api_heavy():
    """API heavy traffic - more database and cache calls"""
    host = random.choice(['api-service-1', 'api-service-2'])
    endpoint = random.choice([
        '/api/db/query',
        '/api/db/users',
        '/api/cache/get/test',
        '/api/cache/set/test/value',
        '/api/simulate/load',
    ])
    make_request(host, endpoint)


def traffic_pattern_gateway():
    """Gateway focused traffic"""
    make_request('nginx-gateway', random.choice(ENDPOINTS), port=80)


def traffic_pattern_queue():
    """Queue traffic - publish messages"""
    host = random.choice(['api-service-1', 'api-service-2'])
    data = {
        'action': random.choice(['process', 'notify', 'analyze']),
        'id': random.randint(1, 10000),
        'timestamp': time.time(),
        'source': 'traffic-generator'
    }
    post_request(host, '/api/queue/publish', data)


def main():
    """Main traffic generation loop"""
    hostname = socket.gethostname()
    logger.info(f"Traffic Generator started on {hostname}")
    logger.info(f"Targets: {TARGET_HOSTS}")
    logger.info(f"Interval: {INTERVAL}s, Concurrent: {CONCURRENT}")

    # Wait for services to be ready
    logger.info("Waiting for services to be ready...")
    time.sleep(10)

    patterns = [
        (traffic_pattern_normal, 0.4),      # 40% normal
        (traffic_pattern_api_heavy, 0.3),   # 30% API heavy
        (traffic_pattern_gateway, 0.2),     # 20% gateway
        (traffic_pattern_queue, 0.1),       # 10% queue
    ]

    iteration = 0
    while True:
        try:
            iteration += 1

            # Select traffic pattern based on weights
            r = random.random()
            cumulative = 0
            for pattern_fn, weight in patterns:
                cumulative += weight
                if r <= cumulative:
                    pattern_fn()
                    break

            # Occasional Redis direct operations
            if random.random() < 0.2:
                redis_operations()

            # Occasional burst traffic (every ~50 iterations)
            if iteration % 50 == 0:
                target = random.choice(TARGET_HOSTS)
                threading.Thread(target=generate_burst_traffic, args=(target, 20)).start()

            # Inter-service calls (every ~10 iterations)
            if iteration % 10 == 0:
                host = random.choice(['api-service-1', 'api-service-2'])
                target = 'api-service-2' if host == 'api-service-1' else 'api-service-1'
                make_request(host, f'/api/call/{target}')

            # Sleep with some jitter
            jitter = random.uniform(-0.5, 0.5)
            time.sleep(max(0.5, INTERVAL + jitter))

        except KeyboardInterrupt:
            logger.info("Shutting down traffic generator")
            break
        except Exception as e:
            logger.error(f"Error in main loop: {e}")
            time.sleep(5)


if __name__ == '__main__':
    main()
