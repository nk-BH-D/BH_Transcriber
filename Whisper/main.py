from contextlib import asynccontextmanager
from fastapi import FastAPI, UploadFile, File, HTTPException
from faster_whisper import WhisperModel
import os
import tempfile
import logging
import asyncio



logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

CONFIG = {
    "whisper": {
        "model_size": str,
        "device": str,
        "compute_type": str,
        "num_workers": int,
    },
    "concurent": {
        "max_concurrent": int
    },
    "MUS": {
        "mus": int
    }
}

def config_loader():
    # Загружаем конфигурацию из переменных окружения
    model_size = os.getenv("WHISPER_MODEL_SIZE")
    if model_size is None:
        raise ValueError("WHISPER_MODEL_SIZE environment variable is not set")
    device = os.getenv("WHISPER_DEVICE")
    if device is None:
        raise ValueError("WHISPER_DEVICE environment variable is not set")
    compute_type = os.getenv("WHISPER_COMPUTE_TYPE")
    if compute_type is None:
        raise ValueError("WHISPER_COMPUTE_TYPE environment variable is not set")
    num_workers = os.getenv("WHISPER_NUM_WORKERS")
    if num_workers is None:
        raise ValueError("WHISPER_NUM_WORKERS environment variable is not set")
    max_concurrent = os.getenv("WHISPER_MAX_CONCURRENT")
    if max_concurrent is None:
        raise ValueError("WHISPER_MAX_CONCURRENT environment variable is not set")
    mus = os.getenv("MAX_UPLOAD_SIZE")
    if mus is None:
        raise ValueError("MAX_UPLOAD_SIZE environment variable is not set")

    
    CONFIG["whisper"]["model_size"] = model_size
    CONFIG["whisper"]["device"] = device
    CONFIG["whisper"]["compute_type"] = compute_type
    CONFIG["whisper"]["num_workers"] = int(num_workers)
    CONFIG["concurent"]["max_concurrent"] = int(max_concurrent)
    CONFIG["MUS"]["mus"] = int(mus) *1024**2  # Convert MB to bytes

    logger.info(f"Configuration loaded: {CONFIG}")
    


@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info("Whisper Service is starting up...")
    config_loader()

    app.state.model = await asyncio.to_thread(
        WhisperModel,
        model_size_or_path=CONFIG["whisper"]["model_size"],
        device=CONFIG["whisper"]["device"],
        compute_type=CONFIG["whisper"]["compute_type"],
        num_workers=CONFIG["whisper"]["num_workers"],
    )
    app.state.transcribe_semaphore = asyncio.Semaphore(CONFIG["concurent"]["max_concurrent"])

    try:
        yield
    finally:
        logger.info("Whisper Service is shutting down...")
        app.state.model = None

app = FastAPI(title="Whisper Service", lifespan=lifespan)


@app.post("/whisper")
async def whisper_endpoint(file: UploadFile = File(...)):
    suffix = os.path.splitext(file.filename or "")[1] or ".audio"
    temp_path = None
    uploaded_bytes = 0

    try:
        with tempfile.NamedTemporaryFile(
            suffix=suffix, delete=False
        ) as temp:
            temp_path = temp.name

            while chunk := await file.read(1024 * 1024):
                uploaded_bytes += len(chunk)
                if uploaded_bytes > CONFIG["MUS"]["mus"]:
                    logger.warning(f"Uploaded file size {uploaded_bytes} exceeds the limit of {CONFIG['MUS']['mus']} bytes")
                    raise HTTPException(
                        status_code=413,
                        detail="Файл превышает допустимый размер",
                    )
                temp.write(chunk)

        async with app.state.transcribe_semaphore:
            text, language, probability = await asyncio.to_thread(
                transcribe_file,
                app.state.model,
                temp_path,
            )

        return {
            "text": text,
            "language": language,
            "language_probability": probability,
        }

    except HTTPException:
        raise
    except Exception as exc:
        logger.exception("Transcription failed")
        raise HTTPException(
            status_code=500,
            detail="Ошибка распознавания",
        ) from exc
    finally:
        if temp_path is not None:
            try:
                os.remove(temp_path)
            except OSError:
                logger.exception("Failed to remove temporary audio file")

        await file.close()


def transcribe_file(model, path):
    segments, info = model.transcribe(path, beam_size=5)
    text = "".join(segment.text for segment in segments).strip()
    return text, info.language, info.language_probability