import sys
import json
from cvx_agent.agents import build_cv_and_letter

def main():
    try:
        # read input JSON from stdin
        input_json = json.load(sys.stdin)

        # call the AI agent
        output = build_cv_and_letter(input_json)

        # write result JSON to stdout
        json.dump(output, sys.stdout, indent=2)

    except Exception as e:
        # fail fast
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
