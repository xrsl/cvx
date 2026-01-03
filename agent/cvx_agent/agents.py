import os
import json
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
# Note: Caching is now handled by the Go layer (cvx build)
# The Python agent is stateless and does not cache results

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
# Main function
# ------------------------
def build_cv_and_letter(input_data: dict, max_retries=2) -> dict:
    """
    input_data: dict with keys:
      - job_posting
      - cv (may be partial)
      - letter (may be partial)
    returns: dict validated against Model

    Note: This function is stateless. Caching is handled by the Go layer.
    """
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
                # Include None values to ensure cv and letter fields are always present
                output_dict = result.output.model_dump(exclude_none=False)
                return output_dict
            except Exception as e:
                print(f"Attempt {attempt+1} failed with {model_name}: {e}")

    raise RuntimeError("All AI attempts failed")
