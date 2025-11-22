"""
API Service - Sample Flask Application for Topology Testing

This service simulates a typical backend API that:
- Receives HTTP requests
- Queries Redis cache
- Queries PostgreSQL database
- Publishes messages to RabbitMQ
"""

import os
import time
import json
import random
import socket
from flask import Flask, jsonify, request

app = Flask(__name__)

# Configuration from environment
SERVICE_NAME = os.environ.get('SERVICE_NAME', 'api-unknown')
SERVICE_PORT = int(os.environ.get('SERVICE_PORT', 8080))
REDIS_HOST = os.environ.get('REDIS_HOST', 'redis-cache')
REDIS_PORT = int(os.environ.get('REDIS_PORT', 6379))
DB_HOST = os.environ.get('DB_HOST', 'postgres-db')
DB_PORT = int(os.environ.get('DB_PORT', 5432))
RABBITMQ_HOST = os.environ.get('RABBITMQ_HOST', 'rabbitmq')

# Lazy connections
redis_client = None
db_conn = None
mq_channel = None


def get_redis():
    """Get Redis connection (lazy initialization)"""
    global redis_client
    if redis_client is None:
        try:
            import redis
            redis_client = redis.Redis(
                host=REDIS_HOST,
                port=REDIS_PORT,
                decode_responses=True
            )
        except Exception as e:
            app.logger.error(f"Redis connection failed: {e}")
    return redis_client


def get_db():
    """Get PostgreSQL connection (lazy initialization)"""
    global db_conn
    if db_conn is None:
        try:
            import psycopg2
            db_conn = psycopg2.connect(
                host=DB_HOST,
                port=DB_PORT,
                user='testuser',
                password='testpass',
                database='testdb'
            )
        except Exception as e:
            app.logger.error(f"Database connection failed: {e}")
    return db_conn


def publish_message(queue, message):
    """Publish message to RabbitMQ"""
    global mq_channel
    try:
        import pika
        if mq_channel is None:
            credentials = pika.PlainCredentials('testuser', 'testpass')
            connection = pika.BlockingConnection(
                pika.ConnectionParameters(host=RABBITMQ_HOST, credentials=credentials)
            )
            mq_channel = connection.channel()
            mq_channel.queue_declare(queue=queue, durable=True)

        mq_channel.basic_publish(
            exchange='',
            routing_key=queue,
            body=json.dumps(message)
        )
        return True
    except Exception as e:
        app.logger.error(f"RabbitMQ publish failed: {e}")
        mq_channel = None
        return False


@app.route('/health')
def health():
    """Health check endpoint"""
    return jsonify({
        'status': 'healthy',
        'service': SERVICE_NAME,
        'hostname': socket.gethostname(),
        'timestamp': time.time()
    })


@app.route('/api/info')
def info():
    """Service information"""
    return jsonify({
        'service': SERVICE_NAME,
        'hostname': socket.gethostname(),
        'ip': socket.gethostbyname(socket.gethostname()),
        'port': SERVICE_PORT,
        'version': '1.0.0',
        'dependencies': {
            'redis': f"{REDIS_HOST}:{REDIS_PORT}",
            'database': f"{DB_HOST}:{DB_PORT}",
            'rabbitmq': RABBITMQ_HOST
        }
    })


@app.route('/api/cache/get/<key>')
def cache_get(key):
    """Get value from Redis cache"""
    redis = get_redis()
    if redis:
        try:
            value = redis.get(key)
            return jsonify({
                'key': key,
                'value': value,
                'source': 'redis',
                'service': SERVICE_NAME
            })
        except Exception as e:
            return jsonify({'error': str(e)}), 500
    return jsonify({'error': 'Redis not available'}), 503


@app.route('/api/cache/set/<key>/<value>')
def cache_set(key, value):
    """Set value in Redis cache"""
    redis = get_redis()
    if redis:
        try:
            redis.set(key, value, ex=300)  # 5 minute expiry
            return jsonify({
                'key': key,
                'value': value,
                'status': 'stored',
                'service': SERVICE_NAME
            })
        except Exception as e:
            return jsonify({'error': str(e)}), 500
    return jsonify({'error': 'Redis not available'}), 503


@app.route('/api/db/query')
def db_query():
    """Execute sample database query"""
    conn = get_db()
    if conn:
        try:
            cursor = conn.cursor()
            cursor.execute("SELECT current_timestamp, current_database()")
            row = cursor.fetchone()
            cursor.close()
            return jsonify({
                'timestamp': str(row[0]),
                'database': row[1],
                'service': SERVICE_NAME
            })
        except Exception as e:
            return jsonify({'error': str(e)}), 500
    return jsonify({'error': 'Database not available'}), 503


@app.route('/api/db/users')
def db_users():
    """Get users from database"""
    conn = get_db()
    if conn:
        try:
            cursor = conn.cursor()
            cursor.execute("SELECT id, name, email FROM users LIMIT 10")
            rows = cursor.fetchall()
            cursor.close()
            return jsonify({
                'users': [{'id': r[0], 'name': r[1], 'email': r[2]} for r in rows],
                'count': len(rows),
                'service': SERVICE_NAME
            })
        except Exception as e:
            return jsonify({'error': str(e)}), 500
    return jsonify({'error': 'Database not available'}), 503


@app.route('/api/queue/publish', methods=['POST'])
def queue_publish():
    """Publish message to RabbitMQ"""
    data = request.get_json() or {}
    message = {
        'data': data,
        'source': SERVICE_NAME,
        'timestamp': time.time()
    }

    if publish_message('tasks', message):
        return jsonify({
            'status': 'published',
            'queue': 'tasks',
            'service': SERVICE_NAME
        })
    return jsonify({'error': 'Queue not available'}), 503


@app.route('/api/simulate/load')
def simulate_load():
    """Simulate CPU load and multiple backend calls"""
    # Simulate processing
    start = time.time()

    results = {
        'service': SERVICE_NAME,
        'operations': []
    }

    # Cache operation
    redis = get_redis()
    if redis:
        try:
            key = f"load_test_{random.randint(1, 1000)}"
            redis.set(key, "test_value", ex=60)
            redis.get(key)
            results['operations'].append({'type': 'cache', 'status': 'success'})
        except:
            results['operations'].append({'type': 'cache', 'status': 'failed'})

    # Database operation
    conn = get_db()
    if conn:
        try:
            cursor = conn.cursor()
            cursor.execute("SELECT COUNT(*) FROM users")
            cursor.fetchone()
            cursor.close()
            results['operations'].append({'type': 'database', 'status': 'success'})
        except:
            results['operations'].append({'type': 'database', 'status': 'failed'})

    # Queue operation
    if publish_message('tasks', {'action': 'load_test'}):
        results['operations'].append({'type': 'queue', 'status': 'success'})
    else:
        results['operations'].append({'type': 'queue', 'status': 'failed'})

    results['duration_ms'] = (time.time() - start) * 1000
    return jsonify(results)


@app.route('/api/call/<target>')
def call_service(target):
    """Call another service (for testing inter-service communication)"""
    import requests
    try:
        resp = requests.get(f"http://{target}:8080/health", timeout=5)
        return jsonify({
            'target': target,
            'status': resp.status_code,
            'response': resp.json(),
            'caller': SERVICE_NAME
        })
    except Exception as e:
        return jsonify({
            'target': target,
            'error': str(e),
            'caller': SERVICE_NAME
        }), 503


if __name__ == '__main__':
    app.run(host='0.0.0.0', port=SERVICE_PORT, debug=True)
