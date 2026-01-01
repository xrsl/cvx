import os
import json
import hashlib
from pathlib import Path
from dotenv import load_dotenv

from pydantic_ai import Agent
from cvx_agent.models import Model
from pydantic_ai.models.google import GoogleModel
from pydantic_ai.models.openai import OpenAIChatModel
from pydantic_ai.models.anthropic import AnthropicModel  # Antropic

load_dotenv()

# ------------------------
# Configuration
# ------------------------
CACHE_DIR = Path(os.getenv("CVX_AGENT_CACHE", ".cache"))
CACHE_DIR.mkdir(exist_ok=True)

PRIMARY_MODEL = os.getenv("AI_MODEL", "claude-haiku-4-5-20251001")
FALLBACK_MODEL = os.getenv("AI_FALLBACK_MODEL", "gemini-2.5-flash")  # could be Claude too

# ------------------------
# Agent helper
# ------------------------
def get_agent(model_name: str) -> Agent:
    """
    Return an Agent object for a given model name.
    Supports Google, OpenAI, and Anthropic.
    """
    model_name_lower = model_name.lower()
    if "gemini" in model_name_lower:
        model = GoogleModel(model_name)
    elif "claude" in model_name_lower:
        model = AnthropicModel(model_name)
    else:
        model = OpenAIChatModel(model_name)

    return Agent(
        model=model,
        system_prompt="""
You are a structured CV and letter generator.
Input is a JSON object containing:
  - job_posting
  - current cv
  - current letter
Return ONLY JSON conforming to the Pydantic Model 'Model'.
Do not include extra text or explanations.
"""
    )

# ------------------------
# Caching helpers
# ------------------------
def hash_input(input_data: dict) -> str:
    s = json.dumps(input_data, sort_keys=True)
    return hashlib.sha256(s.encode()).hexdigest()

def load_cache(hash_key: str) -> dict | None:
    path = CACHE_DIR / f"{hash_key}.json"
    if path.exists():
        return json.loads(path.read_text())
    return None

def save_cache(hash_key: str, data: dict):
    path = CACHE_DIR / f"{hash_key}.json"
    path.write_text(json.dumps(data, indent=2))

# ------------------------
# Main function
# ------------------------
def build_cv_and_letter(input_data: dict, max_retries=2) -> dict:
    """
    input_data: dict with keys:
      - job_posting
      - cv (may be partial)
      - letter (may be partial)
    returns: dict validated against Model
    """
    key = hash_input(input_data)

    # Return cached result if exists
    cached = load_cache(key)
    if cached:
        return cached

    # Try primary and fallback providers
    for model_name in [PRIMARY_MODEL, FALLBACK_MODEL]:
        agent = get_agent(model_name)
        prompt = (
            "Here is the structured input JSON:\n"
            f"{json.dumps(input_data, indent=2)}\n\n"
            "Produce valid JSON output matching the Pydantic Model 'Model'."
        )

        for attempt in range(max_retries):
            try:
                result = agent.run_sync(
                    user_prompt=prompt,
                    output_type=Model
                )
                output_dict = result.output.model_dump()
                save_cache(key, output_dict)
                return output_dict
            except Exception as e:
                print(f"Attempt {attempt+1} failed with {model_name}: {e}")

    raise RuntimeError("All AI attempts failed")
