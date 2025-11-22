# Traffic Generator - Continuous HTTP Traffic Generator
FROM python:3.11-slim

WORKDIR /app

# Install dependencies
RUN pip install --no-cache-dir \
    requests==2.31.0 \
    redis==5.0.1

# Copy script
COPY traffic_gen.py /app/

CMD ["python", "traffic_gen.py"]
