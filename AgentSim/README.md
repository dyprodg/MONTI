# AgentSim

Agent simulation service for generating and managing fake call center agent data.

## Purpose

Simulates 2000+ agents with realistic data patterns including:
- Agent states (available, busy, on call, break, offline)
- Team assignments
- Location assignments
- Real-time status updates

## Rules

### Code Organization
- All simulation logic must be in this directory
- No business logic should leak into other services
- Use clear separation between data generation and data streaming

### Data Generation
- Agents must have realistic state transitions
- Support configurable number of agents (default: 2000)
- Generate agents with varied teams and locations
- Ensure reproducible randomness with seed support

### Performance
- Must handle 2000+ agents without performance degradation
- State updates should be batched when possible
- Memory footprint should remain constant regardless of agent count

### Configuration
- All simulation parameters must be configurable via environment variables or config files
- Default values should work out of the box

### Testing
- Unit tests required for all data generation logic
- Performance tests for large agent counts (2000+)
- Integration tests with Backend service

## Dependencies

- Should have minimal external dependencies
- Can depend on common libraries for random generation
- Must not depend on Backend or Frontend code

## CI/CD

This service will only be built and tested when:
- Files in `AgentSim/` directory are changed
- Root configuration files affecting all services are changed
- Explicitly triggered via workflow dispatch
