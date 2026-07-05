"""Gradio dev UI for local digital twin validation."""

from __future__ import annotations

import os
import sys

from dotenv import load_dotenv

from twin import BASE_DIR, DEFAULT_PERSONA, DigitalTwin, smoke_test

load_dotenv(BASE_DIR / ".env", override=True)


if __name__ == "__main__":
    persona = os.getenv("PERSONA", DEFAULT_PERSONA)
    if len(sys.argv) > 1 and sys.argv[1] == "--smoke-test":
        smoke_test(persona)
        sys.exit(0)

    import gradio as gr

    twin = DigitalTwin(persona)
    gr.ChatInterface(
        twin.chat,
        title=f"Neon AI Cloud — {twin.name}",
        description=f"Digital twin ({persona.upper()}) for local validation.",
    ).launch()
