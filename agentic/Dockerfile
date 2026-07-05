# Production image for the Neon AI Cloud digital twin API (FastAPI / uvicorn).
FROM python:3.11-slim-bookworm AS runtime

WORKDIR /app

RUN groupadd --gid 65532 twin && \
    useradd --uid 65532 --gid twin --create-home --shell /usr/sbin/nologin twin

COPY requirements-api.txt .
RUN pip install --no-cache-dir -r requirements-api.txt

COPY --chown=65532:twin twin.py api.py llm.py ./
COPY --chown=65532:twin ceo cto engineering sales ./

ENV TWIN_HTTP_HOST=0.0.0.0 \
    TWIN_HTTP_PORT=7861 \
    PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1

USER 65532:65532

EXPOSE 7861

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:7861/health', timeout=3)"

CMD ["python", "api.py"]
