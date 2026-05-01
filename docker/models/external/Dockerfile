FROM python:3.10.20-slim-trixie AS fasttext-predict-builder

ARG TARGETPLATFORM
WORKDIR /opt
RUN --mount=type=cache,target=/var/cache/apt,id=apt-cache-${TARGETPLATFORM},sharing=locked \
    --mount=type=cache,target=/var/lib/apt,id=apt-lib-${TARGETPLATFORM},sharing=locked \
    apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates
RUN --mount=type=cache,target=/root/.cache/pip,id=pip-cache-${TARGETPLATFORM} \
    pip wheel --no-cache-dir --wheel-dir /tmp/wheels fasttext-predict

FROM python:3.10.20-slim-trixie

ARG TARGETPLATFORM
RUN --mount=type=cache,target=/var/cache/apt,id=apt-cache-${TARGETPLATFORM},sharing=locked \
    --mount=type=cache,target=/var/lib/apt,id=apt-lib-${TARGETPLATFORM},sharing=locked \
    apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl
RUN mkdir /app
WORKDIR /app

COPY --from=fasttext-predict-builder /tmp/wheels /tmp/wheels
RUN --mount=type=cache,target=/root/.cache/pip,id=pip-cache-${TARGETPLATFORM} \
    pip install --no-cache-dir /tmp/wheels/fasttext_predict-*.whl \
    && rm -rf /tmp/wheels

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
RUN mkdir -p /app/fasttext \
    && curl --fail --show-error --silent --location \
        https://dl.fbaipublicfiles.com/fasttext/supervised-models/lid.176.bin \
        --output /app/fasttext/lid.176.bin
COPY external external

ENV VERSION="0.1.0"

CMD ["python", "-m", "external"]
