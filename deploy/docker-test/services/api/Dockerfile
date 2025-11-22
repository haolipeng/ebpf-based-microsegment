# API Service - Python Flask Application
FROM python:3.11-slim

WORKDIR /app

# Install dependencies
RUN pip install --no-cache-dir \
    flask==3.0.0 \
    redis==5.0.1 \
    psycopg2-binary==2.9.9 \
    pika==1.3.2 \
    requests==2.31.0 \
    gunicorn==21.2.0

# Copy application
COPY app.py /app/

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run application
CMD ["gunicorn", "--bind", "0.0.0.0:8080", "--workers", "2", "app:app"]
