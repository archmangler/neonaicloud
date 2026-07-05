from __future__ import annotations

import json
import os
from pathlib import Path

import requests
from openai import OpenAI
from pypdf import PdfReader

BASE_DIR = Path(__file__).resolve().parent
MODEL = os.getenv("OPENAI_MODEL", "gpt-4o-mini")
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
        self._openai = None
        self.name, self.summary, self.documents = load_persona_assets(persona_id)
        self.role_prompt = PERSONA_PROMPTS.get(
            persona_id,
            f"You are acting as {self.name} on the Neon AI Cloud website.",
        )

    @property
    def openai(self):
        if self._openai is None:
            api_key = os.getenv("OPENAI_API_KEY")
            if not api_key:
                raise RuntimeError(
                    "OPENAI_API_KEY is not set. Copy agentic/.env.example to agentic/.env "
                    "and add your key there, or export OPENAI_API_KEY in the shell before starting "
                    "the server. The key stays server-side only — it is never sent to the chat UI."
                )
            self._openai = OpenAI(api_key=api_key)
        return self._openai

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

    def system_prompt(self) -> str:
        document_blocks = "\n\n".join(
            f"## {label.replace('-', ' ').title()}\n{text}"
            for label, text in self.documents.items()
        )

        return (
            f"You are acting as {self.name}. {self.role_prompt} "
            f"Your responsibility is to represent {self.name} faithfully for website interactions. "
            "Use the summary and reference documents below to answer questions about background, "
            "capabilities, products, and engagement fit. "
            "Be professional and engaging, as if talking to a prospective client evaluating Neon AI Cloud. "
            "If you don't know the answer, use record_unknown_question to record it. "
            "When the user is engaged, steer toward getting in touch via email and record it with record_user_details.\n\n"
            f"## Summary\n{self.summary}\n\n"
            f"## Reference Documents\n{document_blocks}\n\n"
            f"With this context, chat with the user while staying in character as {self.name}."
        )

    def chat(self, message: str, history: list[dict]) -> str:
        messages = [{"role": "system", "content": self.system_prompt()}] + history + [{"role": "user", "content": message}]
        done = False
        while not done:
            response = self.openai.chat.completions.create(model=MODEL, messages=messages, tools=tools)
            if response.choices[0].finish_reason == "tool_calls":
                message_obj = response.choices[0].message
                tool_calls = message_obj.tool_calls
                results = self.handle_tool_call(tool_calls)
                messages.append(message_obj)
                messages.extend(results)
            else:
                done = True
        content = response.choices[0].message.content
        return content or ""


def smoke_test(persona_id: str = DEFAULT_PERSONA) -> None:
    twin = DigitalTwin(persona_id)
    print(f"persona: {twin.persona_id}")
    print(f"name: {twin.name}")
    print(f"summary chars: {len(twin.summary)}")
    for label, text in twin.documents.items():
        print(f"  pdf {label}: {len(text)} chars")
    print(f"system prompt chars: {len(twin.system_prompt())}")
