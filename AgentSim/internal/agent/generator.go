package agent

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/dennisdiepolder/monti/agentsim/internal/types"
)

// Generator creates and manages fake agents
type Generator struct {
	agents []types.Agent
	rng    *rand.Rand
}

// NewGenerator creates a new agent generator
func NewGenerator(seed int64) *Generator {
	return &Generator{
		agents: make([]types.Agent, 0, 200),
		rng:    rand.New(rand.NewSource(seed)),
	}
}

// GenerateAgents creates 200 fake agents with realistic distributions
func (g *Generator) GenerateAgents(count int) []types.Agent {
	departments := []types.Department{
		types.DeptSales,
		types.DeptSupport,
		types.DeptTechnical,
		types.DeptRetention,
	}

	locations := []types.Location{
		types.LocationBerlin,
		types.LocationMunich,
		types.LocationHamburg,
		types.LocationFrankfurt,
		types.LocationRemote,
	}

	// Distribution: 30% Sales, 35% Support, 20% Technical, 15% Retention
	deptWeights := []int{30, 35, 20, 15}

	// Distribution: 25% Berlin, 20% Munich, 15% Hamburg, 15% Frankfurt, 25% Remote
	locWeights := []int{25, 20, 15, 15, 25}

	g.agents = make([]types.Agent, count)

	for i := 0; i < count; i++ {
		dept := g.weightedChoice(departments, deptWeights).(types.Department)
		loc := g.weightedChoice(locations, locWeights).(types.Location)
		team := g.generateTeamName(dept, i)

		g.agents[i] = types.Agent{
			ID:         fmt.Sprintf("AGT-%05d", i+1),
			Department: dept,
			Location:   loc,
			Team:       team,
			State:      types.StateOffline,
			StateStart: time.Now(),
			LastUpdate: time.Now(),
		}
	}

	return g.agents
}

// GetAgents returns all generated agents
func (g *Generator) GetAgents() []types.Agent {
	return g.agents
}

// GetAgentByID returns a specific agent by ID
func (g *Generator) GetAgentByID(id string) *types.Agent {
	for i := range g.agents {
		if g.agents[i].ID == id {
			return &g.agents[i]
		}
	}
	return nil
}

// weightedChoice selects an item based on weights
func (g *Generator) weightedChoice(items interface{}, weights []int) interface{} {
	totalWeight := 0
	for _, w := range weights {
		totalWeight += w
	}

	choice := g.rng.Intn(totalWeight)
	cumulative := 0

	switch v := items.(type) {
	case []types.Department:
		for i, w := range weights {
			cumulative += w
			if choice < cumulative {
				return v[i]
			}
		}
		return v[0]
	case []types.Location:
		for i, w := range weights {
			cumulative += w
			if choice < cumulative {
				return v[i]
			}
		}
		return v[0]
	}

	return nil
}

// generateTeamName creates a team name based on department
func (g *Generator) generateTeamName(dept types.Department, index int) string {
	teamNum := (index % 10) + 1

	switch dept {
	case types.DeptSales:
		return fmt.Sprintf("Sales-Team-%d", teamNum)
	case types.DeptSupport:
		return fmt.Sprintf("Support-Team-%d", teamNum)
	case types.DeptTechnical:
		return fmt.Sprintf("Tech-Team-%d", teamNum)
	case types.DeptRetention:
		return fmt.Sprintf("Retention-Team-%d", teamNum)
	default:
		return fmt.Sprintf("Team-%d", teamNum)
	}
}
