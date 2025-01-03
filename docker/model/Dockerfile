FROM alpine:3.21 AS weights

RUN apk --no-cache add ca-certificates curl

WORKDIR /opt
COPY weights.sh weights.sh
RUN chmod +x weights.sh
RUN /opt/weights.sh

FROM python:3.10-slim-bookworm

ENV VERSION="0.1.0"

RUN apt-get update && rm -rf /var/lib/apt/lists/*
RUN mkdir /app
WORKDIR /app
COPY requirements.txt requirements.txt
RUN pip install -r requirements.txt
COPY --from=weights /opt/weights/mpnetv2/mpnet-base-v2 /app/mpnet-base-v2
COPY mpnetv2 mpnetv2

CMD ["python", "-m", "mpnetv2"]
