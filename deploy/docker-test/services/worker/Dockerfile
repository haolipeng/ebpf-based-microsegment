# Worker Service - Background Job Processor
FROM python:3.11-slim

WORKDIR /app

# Install dependencies
RUN pip install --no-cache-dir \
    redis==5.0.1 \
    psycopg2-binary==2.9.9 \
    pika==1.3.2 \
    elasticsearch==8.11.0 \
    requests==2.31.0

# Copy application
COPY worker.py /app/

# Run worker
CMD ["python", "worker.py"]
