# Edge Rule Engine

## Overview
Edge Rule Engine is a lightweight, local rule execution system designed to process computer vision state and event payloads on edge devices.

## Quick Start

```bash
# Build the project
make build

# Run the engine
make run

# Clean up
make clean
```

## Configuration
See `config.yaml` for default configuration. You can change database path, action directories, retention hours, etc.

## API Reference
- `POST /{camera}/state` : Submit CV state
- `POST /{camera}/event` : Submit CV event
- `POST /api/v1/rules` : Create rule
- `GET /api/v1/rules` : List rules
- `GET /api/v1/rules/{ruleId}` : Get rule
- `PUT /api/v1/rules/{ruleId}` : Update rule
- `DELETE /api/v1/rules/{ruleId}` : Delete rule
- `GET /health` : Healthcheck

## Example Payloads

State:
```json
{
  "emittedAt": 1690000000,
  "camera": "cam-1",
  "regions": [
    {"name": "door", "occupancy": 1}
  ]
}
```

Rule:
```json
{
  "id": "rule-1",
  "name": "door-open",
  "condition": "state.Camera == 'cam-1'",
  "action": "trigger_alarm"
}
```
