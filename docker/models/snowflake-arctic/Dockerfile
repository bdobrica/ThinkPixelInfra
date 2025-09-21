FROM alpine:3.21 AS weights

RUN apk --no-cache add ca-certificates curl

WORKDIR /opt
COPY weights.sh weights.sh
RUN chmod +x weights.sh
RUN /opt/weights.sh

FROM python:3.10-slim-bookworm

ARG TARGETPLATFORM
RUN --mount=type=cache,target=/var/cache/apt,id=apt-cache-${TARGETPLATFORM} \
    --mount=type=cache,target=/var/lib/apt,id=apt-lib-${TARGETPLATFORM} \
    apt-get update
RUN mkdir /app
WORKDIR /app
COPY requirements.txt requirements.txt
RUN --mount=type=cache,target=/root/.cache/pip,id=pip-cache-${TARGETPLATFORM} \
    pip install -r requirements.txt
RUN --mount=type=cache,target=/root/.cache/pip,id=pip-cache-${TARGETPLATFORM} \
    python -m spacy download en_core_web_sm \
    && python -m spacy download fr_core_news_sm \
    && python -m spacy download de_core_news_sm \
    && python -m spacy download es_core_news_sm \
    && python -m spacy download it_core_news_sm \
    && python -m spacy download ro_core_news_sm
COPY --from=weights /opt/weights/snowflake-arctic/snowflake-arctic-embed-l-v2.0 /app/snowflake-arctic-embed-l-v2.0
COPY --from=weights /opt/weights/fasttext /app/fasttext
COPY snowflake_arctic snowflake_arctic

ENV VERSION="0.1.9"

CMD ["python", "-m", "snowflake_arctic"]
