FROM python:3.11-slim AS builder

WORKDIR /build

COPY requirements.txt .

# ставим python-зависимости
RUN pip install --no-cache-dir -r requirements.txt

FROM python:3.11-slim

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    HF_HOME=/models/huggingface

WORKDIR /app

COPY --from=builder /usr/local /usr/local
COPY . /app/whisper/

CMD ["uvicorn", "whisper.main:app"]