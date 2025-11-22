"""
Worker Service - Background Job Processor

This service:
- Consumes messages from RabbitMQ
- Processes tasks
- Stores results in PostgreSQL
- Indexes data in Elasticsearch
- Updates cache in Redis
"""

import os
import time
import json
import socket
import logging

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# Configuration
SERVICE_NAME = os.environ.get('SERVICE_NAME', 'worker')
REDIS_HOST = os.environ.get('REDIS_HOST', 'redis-cache')
REDIS_PORT = int(os.environ.get('REDIS_PORT', 6379))
DB_HOST = os.environ.get('DB_HOST', 'postgres-db')
DB_PORT = int(os.environ.get('DB_PORT', 5432))
RABBITMQ_HOST = os.environ.get('RABBITMQ_HOST', 'rabbitmq')
ES_HOST = os.environ.get('ES_HOST', 'elasticsearch')


def get_redis():
    """Get Redis connection"""
    import redis
    return redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)


def get_db():
    """Get PostgreSQL connection"""
    import psycopg2
    return psycopg2.connect(
        host=DB_HOST,
        port=DB_PORT,
        user='testuser',
        password='testpass',
        database='testdb'
    )


def get_es():
    """Get Elasticsearch client"""
    from elasticsearch import Elasticsearch
    return Elasticsearch([f"http://{ES_HOST}:9200"])


def process_message(channel, method, properties, body):
    """Process a message from the queue"""
    try:
        message = json.loads(body)
        logger.info(f"Processing message: {message}")

        # Simulate processing
        time.sleep(0.5)

        # Update cache
        try:
            redis = get_redis()
            redis.incr('processed_count')
            redis.set(f"last_processed_{SERVICE_NAME}", json.dumps({
                'message': message,
                'timestamp': time.time(),
                'hostname': socket.gethostname()
            }))
            logger.info("Updated Redis cache")
        except Exception as e:
            logger.error(f"Redis error: {e}")

        # Store in database
        try:
            conn = get_db()
            cursor = conn.cursor()
            cursor.execute(
                "INSERT INTO processed_tasks (data, source, processed_at) VALUES (%s, %s, NOW())",
                (json.dumps(message), message.get('source', 'unknown'))
            )
            conn.commit()
            cursor.close()
            conn.close()
            logger.info("Stored in PostgreSQL")
        except Exception as e:
            logger.error(f"Database error: {e}")

        # Index in Elasticsearch
        try:
            es = get_es()
            es.index(
                index='processed-tasks',
                document={
                    'message': message,
                    'worker': SERVICE_NAME,
                    'hostname': socket.gethostname(),
                    'processed_at': time.strftime('%Y-%m-%dT%H:%M:%SZ')
                }
            )
            logger.info("Indexed in Elasticsearch")
        except Exception as e:
            logger.error(f"Elasticsearch error: {e}")

        # Acknowledge message
        channel.basic_ack(delivery_tag=method.delivery_tag)
        logger.info("Message processed successfully")

    except Exception as e:
        logger.error(f"Error processing message: {e}")
        channel.basic_nack(delivery_tag=method.delivery_tag, requeue=True)


def main():
    """Main worker loop"""
    import pika

    logger.info(f"Starting worker {SERVICE_NAME} on {socket.gethostname()}")

    while True:
        try:
            credentials = pika.PlainCredentials('testuser', 'testpass')
            connection = pika.BlockingConnection(
                pika.ConnectionParameters(
                    host=RABBITMQ_HOST,
                    credentials=credentials,
                    heartbeat=600,
                    blocked_connection_timeout=300
                )
            )
            channel = connection.channel()
            channel.queue_declare(queue='tasks', durable=True)
            channel.basic_qos(prefetch_count=1)
            channel.basic_consume(queue='tasks', on_message_callback=process_message)

            logger.info("Connected to RabbitMQ, waiting for messages...")
            channel.start_consuming()

        except pika.exceptions.AMQPConnectionError as e:
            logger.error(f"RabbitMQ connection error: {e}")
            time.sleep(5)
        except Exception as e:
            logger.error(f"Unexpected error: {e}")
            time.sleep(5)


if __name__ == '__main__':
    main()
