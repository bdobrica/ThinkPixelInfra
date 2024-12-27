FROM python:3.10-slim-bookworm

RUN apt-get update && rm -rf /var/lib/apt/lists/*
RUN mkdir /app
WORKDIR /app
COPY requirements.txt requirements.txt
RUN pip install -r requirements.txt
COPY mpnetv2 mpnetv2

CMD ["python", "-m", "mpnetv2"]
