FROM dgshanee/courier

USER root

RUN apk add --no-cache \
    git \
    python3 \
    py3-pip

RUN python3 -m venv --system-site-packages /opt/venv

ENV VIRTUAL_ENV=/opt/venv
ENV PATH="/opt/venv/bin:$PATH"

WORKDIR /work
