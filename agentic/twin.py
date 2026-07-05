from __future__ import annotations

import json
import os
from pathlib import Path

import requests
from openai import APIConnectionError, APIError, APITimeoutError, RateLimitError
from pypdf import PdfReader

from llm import LLMConfig, create_llm_client, load_llm_config, verify_llm_ready

BASE_DIR = Path(__file__).resolve().parent
DEFAULT_PERSONA = os.getenv("PERSONA", "cto")

PERSONA_PROMPTS = {
    "cto": (
        "You are the Chief Technology Officer of Neon AI Cloud. "
        "Answer as a technical leader focused on architecture, platforms, AI infrastructure, "
        "delivery rigor, and product depth (CapOS, Factory CAD). "
        "Speak with precision and engineering credibility to prospective clients evaluating "
        "whether Neon AI Cloud is the right partner for their programme."
    ),
}


def push(text: str) -> None:
    token = os.getenv("PUSHOVER_TOKEN")
    user = os.getenv("PUSHOVER_USER")
    if not token or not user:
        print(f"[pushover skipped] {text}", flush=True)
        return
    requests.post(
        "https://api.pushover.net/1/messages.json",
        data={"token": token, "user": user, "message": text},
        timeout=10,
    )


def record_user_details(email, name="Name not provided", notes="not provided"):
    push(f"Recording {name} with email {email} and notes {notes}")
    return {"recorded": "ok"}


def record_unknown_question(question):
    push(f"Recording unknown question: {question}")
    return {"recorded": "ok"}


record_user_details_json = {
    "name": "record_user_details",
    "description": "Use this tool to record that a user is interested in being in touch and provided an email address",
    "parameters": {
        "type": "object",
        "properties": {
            "email": {
                "type": "string",
                "description": "The email address of this user",
            },
            "name": {
                "type": "string",
                "description": "The user's name, if they provided it",
            },
            "notes": {
                "type": "string",
                "description": "Any additional information about the conversation that's worth recording to give context",
            },
        },
        "required": ["email"],
        "additionalProperties": False,
    },
}

record_unknown_question_json = {
    "name": "record_unknown_question",
    "description": "Always use this tool to record any question that couldn't be answered as you didn't know the answer",
    "parameters": {
        "type": "object",
        "properties": {
            "question": {
                "type": "string",
                "description": "The question that couldn't be answered",
            },
        },
        "required": ["question"],
        "additionalProperties": False,
    },
}

tools = [
    {"type": "function", "function": record_user_details_json},
    {"type": "function", "function": record_unknown_question_json},
]

NO_TOOL_GUIDANCE = (
    "If you cannot answer a question from the provided context, say so clearly. "
    "When the user is engaged, encourage them to get in touch via email and ask for their email address."
)


def extract_pdf_text(path: Path) -> str:
    reader = PdfReader(str(path))
    chunks: list[str] = []
    for page in reader.pages:
        text = page.extract_text()
        if text:
            chunks.append(text.strip())
    return "\n\n".join(chunks)


def load_persona_assets(persona_id: str) -> tuple[str, str, dict[str, str]]:
    persona_dir = BASE_DIR / persona_id
    if not persona_dir.is_dir():
        raise FileNotFoundError(f"Persona folder not found: {persona_dir}")

    summary_path = persona_dir / "summary.txt"
    if not summary_path.is_file():
        raise FileNotFoundError(f"Missing summary.txt for persona '{persona_id}': {summary_path}")

    summary = summary_path.read_text(encoding="utf-8").strip()
    if not summary:
        raise ValueError(f"summary.txt is empty for persona '{persona_id}'")

    documents: dict[str, str] = {}
    for pdf_path in sorted(persona_dir.glob("*.pdf")):
        text = extract_pdf_text(pdf_path).strip()
        if text:
            documents[pdf_path.stem] = text

    if not documents:
        raise ValueError(f"No readable PDF documents found in {persona_dir}")

    name = summary.split("—", 1)[0].strip() if "—" in summary else summary.split("\n", 1)[0].strip()
    return name, summary, documents


class DigitalTwin:
    def __init__(self, persona_id: str = DEFAULT_PERSONA):
        self.persona_id = persona_id
        self._llm = None
        self._llm_config: LLMConfig | None = None
        self.name, self.summary, self.documents = load_persona_assets(persona_id)
        self.role_prompt = PERSONA_PROMPTS.get(
            persona_id,
            f"You are acting as {self.name} on the Neon AI Cloud website.",
        )

    @property
    def llm_config(self) -> LLMConfig:
        if self._llm_config is None:
            self._llm_config = load_llm_config()
        return self._llm_config

    @property
    def llm(self):
        if self._llm is None:
            self._llm = create_llm_client(self.llm_config)
        return self._llm

    def handle_tool_call(self, tool_calls):
        results = []
        for tool_call in tool_calls:
            tool_name = tool_call.function.name
            arguments = json.loads(tool_call.function.arguments)
            print(f"Tool called: {tool_name}", flush=True)
            tool = globals().get(tool_name)
            result = tool(**arguments) if tool else {}
            results.append(
                {
                    "role": "tool",
                    "content": json.dumps(result),
                    "tool_call_id": tool_call.id,
                }
            )
        return results

    def system_prompt(self, include_tool_guidance: bool = True) -> str:
        document_blocks = "\n\n".join(
            f"## {label.replace('-', ' ').title()}\n{text}"
            for label, text in self.documents.items()
        )

        prompt = (
            f"You are acting as {self.name}. {self.role_prompt} "
            f"Your responsibility is to represent {self.name} faithfully for website interactions. "
            "Use the summary and reference documents below to answer questions about background, "
            "capabilities, products, and engagement fit. "
            "Be professional and engaging, as if talking to a prospective client evaluating Neon AI Cloud."
        )

        if include_tool_guidance and self.llm_config.supports_tools:
            prompt += (
                " If you don't know the answer, use record_unknown_question to record it. "
                "When the user is engaged, steer toward getting in touch via email and record it with record_user_details."
            )
        elif include_tool_guidance:
            prompt += f" {NO_TOOL_GUIDANCE}"

        prompt += (
            f"\n\n## Summary\n{self.summary}\n\n"
            f"## Reference Documents\n{document_blocks}\n\n"
            f"With this context, chat with the user while staying in character as {self.name}."
        )
        return prompt

    def _completion(self, messages: list[dict], use_tools: bool):
        kwargs = {
            "model": self.llm_config.model,
            "messages": messages,
        }
        if use_tools:
            kwargs["tools"] = tools
        try:
            return self.llm.chat.completions.create(**kwargs)
        except APIConnectionError as exc:
            raise RuntimeError(
                f"Cannot reach the LLM service ({self.llm_config.provider}). Check that it is running."
            ) from exc
        except APITimeoutError as exc:
            raise RuntimeError("LLM request timed out. Try a shorter question or a smaller model.") from exc
        except RateLimitError as exc:
            raise RuntimeError("LLM rate limit exceeded. Try again shortly.") from exc
        except APIError as exc:
            message = getattr(exc, "message", None) or str(exc)
            if self.llm_config.provider == "ollama" and "not found" in message.lower():
                raise RuntimeError(
                    f"Ollama model '{self.llm_config.model}' is not installed. "
                    f"Run: ollama pull {self.llm_config.model}"
                ) from exc
            raise RuntimeError(f"LLM request failed: {message}") from exc

    def chat(self, message: str, history: list[dict]) -> str:
        use_tools = self.llm_config.supports_tools
        messages = [{"role": "system", "content": self.system_prompt(include_tool_guidance=True)}] + history + [
            {"role": "user", "content": message}
        ]

        try:
            done = False
            while not done:
                response = self._completion(messages, use_tools=use_tools)
                choice = response.choices[0]
                if use_tools and choice.finish_reason == "tool_calls":
                    message_obj = choice.message
                    tool_calls = message_obj.tool_calls
                    results = self.handle_tool_call(tool_calls)
                    messages.append(message_obj)
                    messages.extend(results)
                else:
                    done = True
            content = response.choices[0].message.content
            return content or ""
        except Exception as exc:
            if not use_tools:
                raise
            print(f"[llm] tool calling failed ({exc}); retrying without tools", flush=True)
            messages = [{"role": "system", "content": self.system_prompt(include_tool_guidance=True)}] + history + [
                {"role": "user", "content": message}
            ]
            response = self._completion(messages, use_tools=False)
            content = response.choices[0].message.content
            return content or ""


def smoke_test(persona_id: str = DEFAULT_PERSONA) -> None:
    twin = DigitalTwin(persona_id)
    config = twin.llm_config
    print(f"persona: {twin.persona_id}")
    print(f"name: {twin.name}")
    print(f"llm provider: {config.provider}")
    print(f"llm model: {config.model}")
    if config.base_url:
        print(f"llm base url: {config.base_url}")
    print(f"tool calling: {config.supports_tools}")
    print(f"summary chars: {len(twin.summary)}")
    for label, text in twin.documents.items():
        print(f"  pdf {label}: {len(text)} chars")
    print(f"system prompt chars: {len(twin.system_prompt())}")
