"""
Unified AI agent for cvx operations.
Supports build, extract, and advise actions using pydantic-ai.
"""
import os
import sys
import json
from typing import Any
from pydantic import BaseModel
from dotenv import load_dotenv

from pydantic_ai import Agent
from pydantic_ai.models.google import GoogleModel
from pydantic_ai.models.openai import OpenAIModel
from pydantic_ai.models.anthropic import AnthropicModel
from pydantic_ai.models.groq import GroqModel

load_dotenv()

# ------------------------
# Configuration
# ------------------------
PRIMARY_MODEL = os.getenv("AI_MODEL", "gemini-2.5-flash")
FALLBACK_MODEL = os.getenv("AI_FALLBACK_MODEL", "gemini-2.5-flash")


# ------------------------
# Dynamic output models
# ------------------------
class ExtractedJob(BaseModel):
    """Dynamic job extraction output - fields vary by schema."""
    model_config = {"extra": "allow"}


class AdviseOutput(BaseModel):
    """Career advice output."""
    analysis: str


# ------------------------
# Agent helper
# ------------------------
def get_model(model_name: str):
    """
    Return a pydantic-ai model for a given model name.
    Supports Google, OpenAI, Anthropic, Groq, and any OpenAI-compatible API.
    """
    model_name_lower = model_name.lower()

    # Check for Groq models (openai/, qwen/, llama, etc.)
    if any(x in model_name_lower for x in ["openai/", "qwen/", "llama", "mixtral", "groq"]):
        return GroqModel(model_name)

    # Check for Gemini models
    if any(x in model_name_lower for x in ["gemini", "flash", "pro"]):
        return GoogleModel(model_name)

    # Check for Claude models
    if any(x in model_name_lower for x in ["claude", "sonnet", "opus", "haiku"]):
        return AnthropicModel(model_name)

    # Default to OpenAI (works with any OpenAI-compatible API)
    return OpenAIModel(model_name)


def run_with_fallback(action_fn, max_retries: int = 2) -> Any:
    """Run an action with primary and fallback models."""
    for model_name in [PRIMARY_MODEL, FALLBACK_MODEL]:
        for attempt in range(max_retries):
            try:
                return action_fn(model_name)
            except Exception as e:
                print(f"Attempt {attempt+1} failed with {model_name}: {e}", file=sys.stderr)

    raise RuntimeError("All AI attempts failed")


# ------------------------
# Build action
# ------------------------
def build_cv_and_letter(input_data: dict) -> dict:
    """
    Build tailored CV and letter.

    input_data: dict with keys:
      - job_posting
      - cv (current cv data)
      - letter (current letter data)

    returns: dict with cv and letter
    """
    # Import Model here to avoid circular imports and allow dynamic regeneration
    from cvx_agent.models import Model

    def do_build(model_name: str) -> dict:
        model = get_model(model_name)
        agent = Agent(
            model=model,
            system_prompt="""You are a structured CV and letter generator.
Input is a JSON object containing:
  - job_posting
  - current cv
  - current letter
Return ONLY JSON conforming to the Pydantic Model 'Model'.
Do not include extra text or explanations."""
        )

        prompt = (
            "Here is the structured input JSON:\n"
            f"{json.dumps(input_data, indent=2)}\n\n"
            "Produce valid JSON output matching the Pydantic Model 'Model'."
        )

        result = agent.run_sync(user_prompt=prompt, output_type=Model)
        return result.output.model_dump(exclude_none=False)

    return run_with_fallback(do_build)


# ------------------------
# Extract action (for add command)
# ------------------------
def extract_job(input_data: dict) -> dict:
    """
    Extract job posting fields.

    input_data: dict with keys:
      - job_text: the job posting content
      - url: the job posting URL
      - schema_prompt: the extraction prompt with field definitions

    returns: dict with extracted fields
    """
    def do_extract(model_name: str) -> dict:
        model = get_model(model_name)
        agent = Agent(
            model=model,
            system_prompt=input_data.get("schema_prompt", "Extract job posting information as JSON.")
        )

        user_prompt = f"Job URL: {input_data.get('url', 'N/A')}\n\nJob posting:\n{input_data.get('job_text', '')}"

        result = agent.run_sync(user_prompt=user_prompt, output_type=dict)
        return result.output

    return run_with_fallback(do_extract)


# ------------------------
# Advise action
# ------------------------
def advise_job_match(input_data: dict) -> dict:
    """
    Analyze job-CV match and provide advice.

    input_data: dict with keys:
      - job_posting: the job posting content
      - cv_content: the current CV content
      - workflow_prompt: the advise workflow prompt
      - context: optional additional context

    returns: dict with analysis
    """
    def do_advise(model_name: str) -> dict:
        model = get_model(model_name)
        agent = Agent(
            model=model,
            system_prompt=input_data.get("workflow_prompt", "Analyze the job-CV match and provide strategic career advice.")
        )

        user_parts = [f"## Job Posting\n{input_data.get('job_posting', '')}"]
        if input_data.get("cv_content"):
            user_parts.append(f"## Current CV\n{input_data.get('cv_content', '')}")
        if input_data.get("context"):
            user_parts.append(f"## Additional Context\n{input_data.get('context', '')}")

        user_prompt = "\n\n".join(user_parts)

        result = agent.run_sync(user_prompt=user_prompt, output_type=str)
        return {"analysis": result.output}

    return run_with_fallback(do_advise)


# ------------------------
# Main dispatcher
# ------------------------
def dispatch(input_data: dict) -> dict:
    """
    Dispatch to the appropriate action based on input.

    input_data must contain an "action" field:
      - "build": build CV and letter
      - "extract": extract job posting fields
      - "advise": analyze job match
    """
    action = input_data.get("action", "build")

    if action == "build":
        return build_cv_and_letter(input_data)
    elif action == "extract":
        return extract_job(input_data)
    elif action == "advise":
        return advise_job_match(input_data)
    else:
        raise ValueError(f"Unknown action: {action}")
