from __future__ import annotations

import os
from dataclasses import dataclass

import requests
from openai import OpenAI

PROVIDER_OPENAI = "openai"
PROVIDER_OLLAMA = "ollama"


@dataclass(frozen=True)
class LLMConfig:
    provider: str
    model: str
    base_url: str | None
    api_key: str | None
    supports_tools: bool


def _normalize_provider(value: str) -> str:
    provider = value.strip().lower()
    if provider in {PROVIDER_OPENAI, PROVIDER_OLLAMA}:
        return provider
    raise RuntimeError(
        f"Unsupported LLM_PROVIDER '{value}'. Use '{PROVIDER_OPENAI}' or '{PROVIDER_OLLAMA}'."
    )


def load_llm_config() -> LLMConfig:
    provider = _normalize_provider(os.getenv("LLM_PROVIDER", PROVIDER_OPENAI))

    if provider == PROVIDER_OLLAMA:
        base_url = os.getenv("OLLAMA_BASE_URL", "http://127.0.0.1:11434/v1").rstrip("/")
        model = os.getenv("OLLAMA_MODEL", "llama3.2")
        api_key = os.getenv("OLLAMA_API_KEY", "ollama")
        supports_tools = os.getenv("OLLAMA_SUPPORTS_TOOLS", "false").lower() in {"1", "true", "yes"}
        return LLMConfig(
            provider=provider,
            model=model,
            base_url=base_url,
            api_key=api_key,
            supports_tools=supports_tools,
        )

    api_key = os.getenv("OPENAI_API_KEY", "").strip()
    if not api_key:
        raise RuntimeError(
            "OPENAI_API_KEY is not set. Copy agentic/.env.example to agentic/.env and add your key, "
            "or set LLM_PROVIDER=ollama to use a local Ollama instance."
        )

    return LLMConfig(
        provider=PROVIDER_OPENAI,
        model=os.getenv("OPENAI_MODEL", "gpt-4o-mini"),
        base_url=None,
        api_key=api_key,
        supports_tools=True,
    )


def create_llm_client(config: LLMConfig) -> OpenAI:
    if config.base_url:
        return OpenAI(base_url=config.base_url, api_key=config.api_key)
    return OpenAI(api_key=config.api_key)


def ollama_api_root(base_url: str) -> str:
    base = base_url.rstrip("/")
    if base.endswith("/v1"):
        return base[:-3]
    return base


def list_ollama_models(config: LLMConfig) -> set[str]:
    if not config.base_url:
        return set()
    root = ollama_api_root(config.base_url)
    response = requests.get(f"{root}/api/tags", timeout=5)
    response.raise_for_status()
    names: set[str] = set()
    for item in response.json().get("models", []):
        name = item.get("name", "")
        if not name:
            continue
        names.add(name)
        if ":" in name:
            names.add(name.split(":", 1)[0])
    return names


def model_available(model: str, available: set[str]) -> bool:
    if model in available:
        return True
    if f"{model}:latest" in available:
        return True
    for name in available:
        if name.startswith(f"{model}:"):
            return True
    return False


def verify_llm_ready(config: LLMConfig) -> None:
    if config.provider != PROVIDER_OLLAMA:
        return

    try:
        models = list_ollama_models(config)
    except requests.RequestException as exc:
        raise RuntimeError(
            f"Cannot reach Ollama at {config.base_url}. Is Ollama running? ({exc})"
        ) from exc

    if model_available(config.model, models):
        return

    sample = ", ".join(sorted(models)[:6])
    raise RuntimeError(
        f"Ollama model '{config.model}' is not installed. "
        f"Run: ollama pull {config.model}"
        + (f"  Available: {sample}" if sample else "")
    )
