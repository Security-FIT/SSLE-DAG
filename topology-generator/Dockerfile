FROM python:3

# Set working directory
WORKDIR /app

# Copy content
COPY . .

# Install all dependencies
RUN pip install --no-cache-dir -r requirements.txt

ENV STORE_ENV_NAME="/volume"

CMD ["python3", "topology-generator.py", "-n", "2", "-d", "node_degree.csv", "-o" "network.json"]