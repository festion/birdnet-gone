FROM python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application files
COPY birdnet_display.py .
COPY cache_builder.py .
COPY species_list.csv .
COPY static/ ./static/

# Create cache directory
RUN mkdir -p static/bird_images_cache

# Build initial cache
RUN python cache_builder.py || echo "Cache build skipped - will build on first run"

# Expose Flask port
EXPOSE 5000

# Run the application
CMD ["python", "birdnet_display.py"]
