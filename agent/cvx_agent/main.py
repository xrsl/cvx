"""Entry point for running cvx_agent as a module."""
import sys
import json
from cvx_agent.agents import dispatch


def main():
    try:
        # read input JSON from stdin
        input_json = json.load(sys.stdin)

        # dispatch to appropriate action
        output = dispatch(input_json)

        # write result JSON to stdout
        json.dump(output, sys.stdout, indent=2)

    except Exception as e:
        # fail fast
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()

# Also support running as a module with -m cvx_agent for backward compatibility
if __name__ == "cvx_agent.__main__":
    main()
