FROM golang:1.26.1

COPY ./requirements.txt /tmp/requirements.txt

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    python3-pip \
    ffmpeg \
    ca-certificates \
    && pip3 install --break-system-packages -r /tmp/requirements.txt \
    && rm -rf /var/lib/apt/lists/* \
    && go install github.com/air-verse/air@latest

WORKDIR /app
