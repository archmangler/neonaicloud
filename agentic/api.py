from __future__ import annotations

import os
from functools import lru_cache

from dotenv import load_dotenv
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

from twin import BASE_DIR, DigitalTwin, smoke_test

load_dotenv(BASE_DIR / ".env", override=True)

app = FastAPI(title="Neon AI Cloud Digital Twin API", version="0.2.0")

ALLOWED_PERSONAS = {"ceo", "cto", "engineering", "sales"}
MAX_MESSAGE_LEN = 4000
MAX_HISTORY = 40


class ChatMessage(BaseModel):
    role: str
    content: str


class ChatRequest(BaseModel):
    message: str = Field(..., min_length=1, max_length=MAX_MESSAGE_LEN)
    history: list[ChatMessage] = Field(default_factory=list)


class ChatResponse(BaseModel):
    reply: str
    persona: str
    name: str


class HealthResponse(BaseModel):
    status: str
    persona: str
    name: str


@lru_cache(maxsize=8)
def get_twin(persona_id: str) -> DigitalTwin:
    return DigitalTwin(persona_id)


def validate_persona(persona_id: str) -> str:
    persona = persona_id.strip().lower()
    if persona not in ALLOWED_PERSONAS:
        raise HTTPException(status_code=404, detail=f"Unknown persona: {persona_id}")
    return persona


def normalize_history(history: list[ChatMessage]) -> list[dict]:
    if len(history) > MAX_HISTORY:
        history = history[-MAX_HISTORY:]
    normalized: list[dict] = []
    for item in history:
        role = item.role.strip().lower()
        if role not in {"user", "assistant"}:
            continue
        content = item.content.strip()
        if not content:
            continue
        normalized.append({"role": role, "content": content[:MAX_MESSAGE_LEN]})
    return normalized


@app.get("/health")
def root_health() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/api/twin/{persona}/health", response_model=HealthResponse)
def persona_health(persona: str) -> HealthResponse:
    persona_id = validate_persona(persona)
    try:
        twin = get_twin(persona_id)
    except FileNotFoundError as exc:
        raise HTTPException(status_code=404, detail=str(exc)) from exc
    except ValueError as exc:
        raise HTTPException(status_code=500, detail=str(exc)) from exc
    return HealthResponse(status="ok", persona=twin.persona_id, name=twin.name)


@app.post("/api/twin/{persona}/chat", response_model=ChatResponse)
def persona_chat(persona: str, body: ChatRequest) -> ChatResponse:
    persona_id = validate_persona(persona)
    message = body.message.strip()
    if not message:
        raise HTTPException(status_code=400, detail="message is required")

    try:
        twin = get_twin(persona_id)
        reply = twin.chat(message, normalize_history(body.history))
    except RuntimeError as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    except FileNotFoundError as exc:
        raise HTTPException(status_code=404, detail=str(exc)) from exc
    except ValueError as exc:
        raise HTTPException(status_code=500, detail=str(exc)) from exc

    return ChatResponse(reply=reply, persona=twin.persona_id, name=twin.name)


if __name__ == "__main__":
    import sys

    import uvicorn

    if len(sys.argv) > 1 and sys.argv[1] == "--smoke-test":
        smoke_test(os.getenv("PERSONA", "cto"))
        sys.exit(0)

    host = os.getenv("TWIN_HTTP_HOST", "127.0.0.1")
    port = int(os.getenv("TWIN_HTTP_PORT", "7861"))
    uvicorn.run("api:app", host=host, port=port, reload=False)
