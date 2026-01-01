"""Tests for the CVX agent."""
import json
import pytest
from cvx_agent.agents import build_cv_and_letter, hash_input


@pytest.fixture
def sample_input():
    """Sample input data for testing."""
    return {
        "job_posting": "We are looking for a Senior Software Engineer...",
        "cv": {
            "name": "John Doe",
            "email": "john@example.com",
            "sections": {
                "experience": [
                    {
                        "company": "Tech Corp",
                        "position": "Software Engineer",
                        "start_date": "2020-01",
                        "end_date": "present",
                    }
                ]
            },
        },
        "letter": {
            "sender": {"name": "John Doe", "email": "john@example.com"},
            "recipient": {"name": "Hiring Manager", "company": "Target Co"},
            "content": {
                "salutation": "Dear Hiring Manager",
                "opening": "I am writing to apply...",
                "closing": "Sincerely",
            },
            "metadata": {"date": "auto"},
        },
    }


def test_hash_input_deterministic(sample_input):
    """Test that hash_input produces consistent hashes."""
    hash1 = hash_input(sample_input)
    hash2 = hash_input(sample_input)
    assert hash1 == hash2
    assert len(hash1) == 64  # SHA256 produces 64 hex characters


def test_hash_input_different_for_different_inputs(sample_input):
    """Test that different inputs produce different hashes."""
    hash1 = hash_input(sample_input)

    modified_input = sample_input.copy()
    modified_input["job_posting"] = "Different job posting"
    hash2 = hash_input(modified_input)

    assert hash1 != hash2


def test_build_cv_and_letter_structure(sample_input):
    """Test that build_cv_and_letter returns correct structure."""
    result = build_cv_and_letter(sample_input)

    # Check top-level keys
    assert "cv" in result
    assert "letter" in result

    # Check CV structure
    assert isinstance(result["cv"], dict)
    assert "name" in result["cv"]
    assert "email" in result["cv"]

    # Check Letter structure
    assert isinstance(result["letter"], dict)
    assert "sender" in result["letter"]
    assert "recipient" in result["letter"]
    assert "content" in result["letter"]


def test_build_cv_and_letter_caching(sample_input):
    """Test that caching works correctly."""
    # The agent uses .cache directory by default
    # First call - should hit AI or cache
    result1 = build_cv_and_letter(sample_input)

    # Second call - should return same result (from cache or AI)
    result2 = build_cv_and_letter(sample_input)

    # Results should be identical
    assert result1 == result2

    # Both should have the same structure
    assert "cv" in result1 and "cv" in result2
    assert "letter" in result1 and "letter" in result2


def test_build_cv_and_letter_with_missing_cv(sample_input):
    """Test handling of missing CV data."""
    sample_input["cv"] = None

    # Should still work - agent will generate from scratch
    result = build_cv_and_letter(sample_input)
    assert "cv" in result
    assert "letter" in result


def test_build_cv_and_letter_with_missing_letter(sample_input):
    """Test handling of missing letter data."""
    sample_input["letter"] = None

    # Should still work - agent will generate from scratch
    result = build_cv_and_letter(sample_input)
    assert "cv" in result
    assert "letter" in result


@pytest.mark.parametrize(
    "model_name",
    [
        "gemini-2.5-flash",
        "claude-haiku-4-5-20251001",
    ],
)
def test_build_with_different_models(sample_input, model_name, monkeypatch):
    """Test that different models work."""
    monkeypatch.setenv("AI_MODEL", model_name)

    # Re-import to pick up new env var
    import importlib
    import cvx_agent.agents

    importlib.reload(cvx_agent.agents)
    from cvx_agent.agents import build_cv_and_letter as build_func

    result = build_func(sample_input)
    assert "cv" in result
    assert "letter" in result


def test_input_validation():
    """Test that the agent handles various input scenarios."""
    # The agent is designed to be resilient and will attempt to process
    # even incomplete input, so we test that it returns valid structure

    # Minimal input with just job posting
    minimal_input = {
        "job_posting": "Software engineer position",
        "cv": {"name": "Test"},
        "letter": {
            "sender": {"name": "Test", "email": "test@example.com"},
            "recipient": {"name": "HR", "company": "Co"},
            "content": {"salutation": "Dear", "opening": "Hi", "closing": "Bye"},
            "metadata": {"date": "auto"}
        }
    }

    result = build_cv_and_letter(minimal_input)

    # Should still return valid structure
    assert "cv" in result
    assert "letter" in result
    assert isinstance(result["cv"], dict)
    assert isinstance(result["letter"], dict)
