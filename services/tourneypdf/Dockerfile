FROM python:3

# Install Noto Serif Font
RUN apt-get update && apt-get install -y \
    fonts-open-sans \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY . /app

RUN pip install --no-cache-dir -r requirements.txt

# Run main.py when the container launches
CMD ["python", "src/main.py"]