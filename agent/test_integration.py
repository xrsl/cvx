"""Integration tests for the CVX agent entry point."""
import json
import subprocess
import sys
import pytest


def test_agent_stdin_stdout():
    """Test that the agent correctly reads from stdin and writes to stdout."""
    input_data = {
        "job_posting": "We are hiring a Python developer with 5+ years of experience.",
        "cv": {
            "name": "Alice Smith",
            "email": "alice@example.com",
            "phone": "+1234567890",
            "sections": {
                "experience": [
                    {
                        "company": "Previous Corp",
                        "position": "Python Developer",
                        "start_date": "2018-01",
                        "end_date": "2023-12",
                        "highlights": ["Built REST APIs", "Optimized database queries"],
                    }
                ],
                "skills": [
                    {"label": "Programming", "details": "Python, Go, JavaScript"}
                ],
            },
        },
        "letter": {
            "sender": {
                "name": "Alice Smith",
                "email": "alice@example.com",
                "phone": "+1234567890",
            },
            "recipient": {"name": "HR Manager", "company": "Target Company"},
            "content": {
                "salutation": "Dear HR Manager",
                "opening": "I am excited to apply for the Python Developer position.",
                "closing": "Best regards",
            },
            "metadata": {"date": "auto"},
        },
    }

    # Call the agent via subprocess using the renamed main.py
    proc = subprocess.Popen(
        ["python", "-m", "cvx_agent.main"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        cwd="/Users/resulal/ghub/cvx/agent",
    )

    stdout, stderr = proc.communicate(input=json.dumps(input_data))

    # Check that it succeeded
    assert proc.returncode == 0, f"Agent failed with stderr: {stderr}"

    # Parse output
    try:
        output = json.loads(stdout)
    except json.JSONDecodeError as e:
        raise AssertionError(f"Invalid JSON output: {stdout}") from e

    # Validate output structure
    assert "cv" in output, "Output missing 'cv' field"
    assert "letter" in output, "Output missing 'letter' field"

    # Validate CV structure
    cv = output["cv"]
    assert isinstance(cv, dict)
    assert "name" in cv
    assert "email" in cv
    assert "sections" in cv

    # Validate Letter structure
    letter = output["letter"]
    assert isinstance(letter, dict)
    assert "sender" in letter
    assert "recipient" in letter
    assert "content" in letter


def test_agent_with_invalid_input():
    """Test that the agent handles incomplete input gracefully."""
    # The agent is resilient and will try to process incomplete input
    # So we test that it at least returns valid JSON structure
    incomplete_input = {
        "job_posting": "Test position",
        "cv": {"name": "Test"},
        "letter": {
            "sender": {"name": "Test", "email": "test@example.com"},
            "recipient": {"name": "HR", "company": "Co"},
            "content": {"salutation": "Dear", "opening": "Hi", "closing": "Bye"},
            "metadata": {"date": "auto"}
        }
    }

    proc = subprocess.Popen(
        ["python", "-m", "cvx_agent.main"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        cwd="/Users/resulal/ghub/cvx/agent",
    )

    stdout, stderr = proc.communicate(input=json.dumps(incomplete_input))

    # Agent should succeed even with minimal input
    assert proc.returncode == 0, f"Agent failed: {stderr}"

    # Should return valid JSON
    try:
        output = json.loads(stdout)
        assert "cv" in output
        assert "letter" in output
    except json.JSONDecodeError:
        pytest.fail(f"Invalid JSON output: {stdout}")


def test_agent_with_malformed_json():
    """Test that the agent handles malformed JSON gracefully."""
    malformed_json = "{ this is not valid json"

    proc = subprocess.Popen(
        ["python", "-m", "cvx_agent.main"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        cwd="/Users/resulal/ghub/cvx/agent",
    )

    stdout, stderr = proc.communicate(input=malformed_json)

    # Should fail with non-zero exit code
    assert proc.returncode != 0, "Agent should fail with malformed JSON"
    assert "Error" in stderr or "error" in stderr.lower(), "Should have error in stderr"
