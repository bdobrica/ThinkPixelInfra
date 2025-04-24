FROM python:3.10-slim-bookworm

WORKDIR /app

# copy and install only what we need
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# copy code + response
COPY server.py ./

# start the server
CMD ["python3", "server.py"]
