"""Tools that the agent can use. Add new tools here to give the agent new capabilities."""


# Tool definitions — sent to the LLM so it knows what it can call.
TOOL_DEFINITIONS = [
    {
        "type": "function",
        "function": {
            "name": "get_weather",
            "description": "Get current weather for a location",
            "parameters": {
                "type": "object",
                "properties": {
                    "location": {"type": "string", "description": "City or location name"}
                },
                "required": ["location"],
            },
        },
    }
]


def execute(name: str, args: dict) -> str:
    """Execute a tool by name and return the result string."""
    location = args.get("location", "unknown")
    if name == "get_weather":
        return f"72°F and sunny in {location}"
    return f"Unknown tool: {name}"
