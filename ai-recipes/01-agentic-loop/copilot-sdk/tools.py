from __future__ import annotations

import httpx
from pydantic import BaseModel, Field

from copilot.tools import define_tool


class LookupWordParams(BaseModel):
    word: str = Field(description="The word to look up")


@define_tool(description="Look up a word's definition in the dictionary")
async def lookup_word(params: LookupWordParams) -> str:
    async with httpx.AsyncClient(timeout=20) as client:
        resp = await client.get(f"https://api.dictionaryapi.dev/api/v2/entries/en/{params.word}")
        if resp.status_code != 200:
            return f"No definition found for '{params.word}'"
        data = resp.json()[0]
        meanings = data.get("meanings", [])
        if not meanings:
            return f"No meanings found for '{params.word}'"
        meaning = meanings[0]
        pos = meaning.get("partOfSpeech", "unknown")
        definition = meaning["definitions"][0]["definition"]
        return f"{params.word} ({pos}): {definition}"


class ConvertUnitsParams(BaseModel):
    value: float = Field(description="The numeric value to convert")
    from_unit: str = Field(description="Source unit (e.g., km, miles, celsius, fahrenheit, kg, lbs)")
    to_unit: str = Field(description="Target unit")


@define_tool(description="Convert between common measurement units")
async def convert_units(params: ConvertUnitsParams) -> str:
    conversions = {
        ("km", "miles"): lambda value: value * 0.621371,
        ("miles", "km"): lambda value: value * 1.60934,
        ("celsius", "fahrenheit"): lambda value: value * 9 / 5 + 32,
        ("fahrenheit", "celsius"): lambda value: (value - 32) * 5 / 9,
        ("kg", "lbs"): lambda value: value * 2.20462,
        ("lbs", "kg"): lambda value: value * 0.453592,
    }
    key = (params.from_unit.lower(), params.to_unit.lower())
    if key not in conversions:
        return f"Cannot convert from {params.from_unit} to {params.to_unit}"
    result = conversions[key](params.value)
    return f"{params.value} {params.from_unit} = {result:.2f} {params.to_unit}"
