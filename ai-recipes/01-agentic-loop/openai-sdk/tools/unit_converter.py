from __future__ import annotations

from typing import Any

TOOL_DEFINITION: dict[str, Any] = {
    'type': 'function',
    'name': 'convert_units',
    'description': 'Convert a numeric value between common units such as miles and kilometers, pounds and kilograms, or Fahrenheit and Celsius.',
    'parameters': {
        'type': 'object',
        'properties': {
            'value': {
                'type': 'number',
                'description': 'The numeric value to convert.',
            },
            'from_unit': {
                'type': 'string',
                'description': 'The unit you are converting from, for example fahrenheit, miles, or kilograms.',
            },
            'to_unit': {
                'type': 'string',
                'description': 'The unit you want to convert to, for example celsius, kilometers, or pounds.',
            },
        },
        'required': ['value', 'from_unit', 'to_unit'],
        'additionalProperties': False,
    },
}

_LENGTH_FACTORS = {
    'meter': 1.0,
    'kilometer': 1000.0,
    'mile': 1609.344,
    'foot': 0.3048,
    'inch': 0.0254,
}
_WEIGHT_FACTORS = {
    'kilogram': 1.0,
    'gram': 0.001,
    'pound': 0.45359237,
    'ounce': 0.028349523125,
}
_VOLUME_FACTORS = {
    'liter': 1.0,
    'milliliter': 0.001,
    'gallon': 3.785411784,
}
_UNIT_ALIASES = {
    'm': 'meter',
    'meter': 'meter',
    'meters': 'meter',
    'metre': 'meter',
    'metres': 'meter',
    'km': 'kilometer',
    'kilometer': 'kilometer',
    'kilometers': 'kilometer',
    'kilometre': 'kilometer',
    'kilometres': 'kilometer',
    'mi': 'mile',
    'mile': 'mile',
    'miles': 'mile',
    'ft': 'foot',
    'foot': 'foot',
    'feet': 'foot',
    'in': 'inch',
    'inch': 'inch',
    'inches': 'inch',
    'kg': 'kilogram',
    'kilogram': 'kilogram',
    'kilograms': 'kilogram',
    'g': 'gram',
    'gram': 'gram',
    'grams': 'gram',
    'lb': 'pound',
    'lbs': 'pound',
    'pound': 'pound',
    'pounds': 'pound',
    'oz': 'ounce',
    'ounce': 'ounce',
    'ounces': 'ounce',
    'l': 'liter',
    'liter': 'liter',
    'liters': 'liter',
    'litre': 'liter',
    'litres': 'liter',
    'ml': 'milliliter',
    'milliliter': 'milliliter',
    'milliliters': 'milliliter',
    'millilitre': 'milliliter',
    'millilitres': 'milliliter',
    'gal': 'gallon',
    'gallon': 'gallon',
    'gallons': 'gallon',
    'c': 'celsius',
    '°c': 'celsius',
    'celsius': 'celsius',
    'degree celsius': 'celsius',
    'degrees celsius': 'celsius',
    'f': 'fahrenheit',
    '°f': 'fahrenheit',
    'fahrenheit': 'fahrenheit',
    'degree fahrenheit': 'fahrenheit',
    'degrees fahrenheit': 'fahrenheit',
    'k': 'kelvin',
    'kelvin': 'kelvin',
}
_UNIT_CATEGORIES = {
    **{unit: 'length' for unit in _LENGTH_FACTORS},
    **{unit: 'weight' for unit in _WEIGHT_FACTORS},
    **{unit: 'volume' for unit in _VOLUME_FACTORS},
    'celsius': 'temperature',
    'fahrenheit': 'temperature',
    'kelvin': 'temperature',
}
_SUPPORTED_UNITS = ', '.join(sorted(_UNIT_ALIASES))


def _normalize_unit(unit: str) -> str:
    normalized = ' '.join(unit.strip().lower().replace('_', ' ').split())
    return _UNIT_ALIASES.get(normalized, '')


def _format_number(value: float) -> str:
    rounded = round(value, 4)
    if float(rounded).is_integer():
        return str(int(rounded))
    return f'{rounded:.4f}'.rstrip('0').rstrip('.')


def _convert_temperature(value: float, from_unit: str, to_unit: str) -> float:
    if from_unit == to_unit:
        return value

    if from_unit == 'celsius':
        celsius_value = value
    elif from_unit == 'fahrenheit':
        celsius_value = (value - 32) * 5 / 9
    else:
        celsius_value = value - 273.15

    if to_unit == 'celsius':
        return celsius_value
    if to_unit == 'fahrenheit':
        return celsius_value * 9 / 5 + 32
    return celsius_value + 273.15


def _convert_scaled(value: float, from_unit: str, to_unit: str, factors: dict[str, float]) -> float:
    return value * factors[from_unit] / factors[to_unit]


def convert_units(value: float, from_unit: str, to_unit: str) -> str:
    source_unit = _normalize_unit(from_unit)
    target_unit = _normalize_unit(to_unit)

    if not source_unit or not target_unit:
        return f'Unsupported unit. Supported units include: {_SUPPORTED_UNITS}.'

    source_category = _UNIT_CATEGORIES[source_unit]
    target_category = _UNIT_CATEGORIES[target_unit]
    if source_category != target_category:
        return f'Cannot convert {source_unit} to {target_unit} because they are different unit types.'

    if source_category == 'temperature':
        converted_value = _convert_temperature(value, source_unit, target_unit)
    elif source_category == 'length':
        converted_value = _convert_scaled(value, source_unit, target_unit, _LENGTH_FACTORS)
    elif source_category == 'weight':
        converted_value = _convert_scaled(value, source_unit, target_unit, _WEIGHT_FACTORS)
    else:
        converted_value = _convert_scaled(value, source_unit, target_unit, _VOLUME_FACTORS)

    return (
        f'{_format_number(value)} {source_unit}'
        f' = {_format_number(converted_value)} {target_unit}'
    )
