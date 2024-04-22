FROM python:3

# RUN apt-get update && apt-get install -y pkg-config

WORKDIR /app
COPY . /app

RUN pip install --no-cache-dir -r requirements.txt

# Run main.py when the container launches
CMD ["python", "src/main.py"]