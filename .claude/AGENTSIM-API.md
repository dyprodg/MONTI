# AgentSim Control API

Base URL: `http://localhost:8081`

## Quick Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/status` | Simulation status |
| POST | `/start` | Start simulation |
| POST | `/stop` | Stop simulation |
| POST | `/scale` | Scale active agents |
| GET | `/config` | Get config |
| GET | `/stats` | Get statistics |
| GET | `/metrics` | Prometheus metrics |
| GET | `/calls/config` | Call generation config |
| PUT | `/calls/config` | Update call gen config |
| GET | `/calls/stats` | Call gen statistics |

## Examples

### Start simulation (100 agents)

```bash
curl -X POST http://localhost:8081/start -d '{"activeAgents":100}'
```

### Stop simulation

```bash
curl -X POST http://localhost:8081/stop
```

### Scale to 500 agents (while running)

```bash
curl -X POST http://localhost:8081/scale -d '{"activeAgents":500}'
```

### Check status

```bash
curl http://localhost:8081/status
```

### View call generation config

```bash
curl http://localhost:8081/calls/config
```

### Increase call volume (peak factor)

```bash
curl -X PUT http://localhost:8081/calls/config -d '{"peakHourFactor":2.0}'
```

### Adjust calls per minute for a department

```bash
curl -X PUT http://localhost:8081/calls/config \
  -d '{"departments":{"sales":{"callsPerMin":5.0},"support":{"callsPerMin":8.0}}}'
```

### View statistics

```bash
curl http://localhost:8081/stats
```

### View call generation stats

```bash
curl http://localhost:8081/calls/stats
```

## Notes

- Max agents: 2000 (configured via `AGENTSIM_AGENTS` env var)
- Default active on start: 100
- Agents simulate 3-30 minute call durations
- Call generation runs independently; adjust `peakHourFactor` and per-department `callsPerMin` to control queue pressure
