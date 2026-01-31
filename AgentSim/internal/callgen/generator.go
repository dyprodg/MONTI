package callgen

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/dennisdiepolder/monti/agentsim/internal/types"
	"github.com/rs/zerolog/log"
)

// DepartmentConfig holds the call generation config for one department.
type DepartmentConfig struct {
	CallsPerMin float64
	VQs         []VQWeight
}

// VQWeight pairs a VQ name with a relative weight for distribution.
type VQWeight struct {
	VQ     types.VQName
	Weight float64
}

// CallGenerator generates calls at configurable rates per department and
// enqueues them via a CallAPIClient.
type CallGenerator struct {
	mu             sync.RWMutex
	departments    map[types.Department]DepartmentConfig
	peakHourFactor float64
	client         *CallAPIClient
}

// NewCallGenerator creates a CallGenerator with default department configs.
func NewCallGenerator(client *CallAPIClient) *CallGenerator {
	g := &CallGenerator{
		peakHourFactor: 1.0,
		client:         client,
		departments:    defaultDepartments(),
	}
	return g
}

func defaultDepartments() map[types.Department]DepartmentConfig {
	return map[types.Department]DepartmentConfig{
		types.DeptSales: {
			CallsPerMin: 40,
			VQs: []VQWeight{
				{VQ: types.VQSalesInbound, Weight: 4},
				{VQ: types.VQSalesOutbound, Weight: 3},
				{VQ: types.VQSalesCallback, Weight: 3},
				{VQ: types.VQSalesChat, Weight: 3},
			},
		},
		types.DeptSupport: {
			CallsPerMin: 60,
			VQs: []VQWeight{
				{VQ: types.VQSupportGeneral, Weight: 4},
				{VQ: types.VQSupportBilling, Weight: 3},
				{VQ: types.VQSupportCallback, Weight: 3},
				{VQ: types.VQSupportChat, Weight: 3},
			},
		},
		types.DeptTechnical: {
			CallsPerMin: 30,
			VQs: []VQWeight{
				{VQ: types.VQTechL1, Weight: 4},
				{VQ: types.VQTechL2, Weight: 3},
				{VQ: types.VQTechCallback, Weight: 3},
				{VQ: types.VQTechChat, Weight: 3},
			},
		},
		types.DeptRetention: {
			CallsPerMin: 20,
			VQs: []VQWeight{
				{VQ: types.VQRetentionSave, Weight: 4},
				{VQ: types.VQRetentionCancel, Weight: 3},
				{VQ: types.VQRetentionCallback, Weight: 3},
				{VQ: types.VQRetentionChat, Weight: 3},
			},
		},
	}
}

// SetDepartmentConfig thread-safely updates the config for a single department.
func (g *CallGenerator) SetDepartmentConfig(dept types.Department, cfg DepartmentConfig) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.departments[dept] = cfg
}

// SetPeakHourFactor thread-safely sets the peak hour multiplier.
// 1.0 = normal rate, 2.0 = double rate.
func (g *CallGenerator) SetPeakHourFactor(factor float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.peakHourFactor = factor
}

// PeakHourFactor returns the current peak hour factor.
func (g *CallGenerator) PeakHourFactor() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.peakHourFactor
}

// Run starts generating calls for all departments until ctx is cancelled.
// It spawns one goroutine per department. The method blocks until all
// goroutines finish (i.e. until the context is done).
func (g *CallGenerator) Run(ctx context.Context) {
	var wg sync.WaitGroup

	g.mu.RLock()
	depts := make([]types.Department, 0, len(g.departments))
	for dept := range g.departments {
		depts = append(depts, dept)
	}
	g.mu.RUnlock()

	for _, dept := range depts {
		wg.Add(1)
		go func(d types.Department) {
			defer wg.Done()
			g.runDepartment(ctx, d)
		}(dept)
	}

	wg.Wait()
}

// runDepartment generates calls for a single department at the configured rate.
func (g *CallGenerator) runDepartment(ctx context.Context, dept types.Department) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(len(dept))))

	for {
		// Read current config under lock.
		g.mu.RLock()
		cfg := g.departments[dept]
		factor := g.peakHourFactor
		g.mu.RUnlock()

		effectiveRate := cfg.CallsPerMin * factor
		if effectiveRate <= 0 {
			// No calls configured; sleep and re-check.
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}

		// Poisson-ish sleep: base interval with jitter.
		baseSleep := time.Duration(float64(time.Minute) / effectiveRate)
		jitter := time.Duration(float64(baseSleep) * (rng.Float64()*0.5 - 0.25)) // +/-25%
		sleep := baseSleep + jitter
		if sleep < time.Millisecond {
			sleep = time.Millisecond
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(sleep):
		}

		// Pick a VQ based on weights.
		vq := pickVQ(rng, cfg.VQs)

		if err := g.client.EnqueueCall(string(vq)); err != nil {
			log.Error().Err(err).
				Str("department", string(dept)).
				Str("vq", string(vq)).
				Msg("failed to enqueue call")
		} else {
			log.Debug().
				Str("department", string(dept)).
				Str("vq", string(vq)).
				Float64("effectiveRate", effectiveRate).
				Msg("enqueued call")
		}
	}
}

// GetDepartmentConfigs returns a copy of the current department configs.
func (g *CallGenerator) GetDepartmentConfigs() map[types.Department]DepartmentConfig {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make(map[types.Department]DepartmentConfig, len(g.departments))
	for k, v := range g.departments {
		out[k] = v
	}
	return out
}

// GetStats returns generation statistics.
func (g *CallGenerator) GetStats() map[string]interface{} {
	g.mu.RLock()
	defer g.mu.RUnlock()
	stats := map[string]interface{}{
		"peakHourFactor": g.peakHourFactor,
		"departments":    map[string]interface{}{},
	}
	deptStats := stats["departments"].(map[string]interface{})
	for dept, cfg := range g.departments {
		vqs := make([]string, 0, len(cfg.VQs))
		for _, v := range cfg.VQs {
			vqs = append(vqs, string(v.VQ))
		}
		deptStats[string(dept)] = map[string]interface{}{
			"callsPerMin": cfg.CallsPerMin,
			"vqs":         vqs,
		}
	}
	return stats
}

// pickVQ selects a VQ based on the configured weights.
func pickVQ(rng *rand.Rand, vqs []VQWeight) types.VQName {
	if len(vqs) == 0 {
		return ""
	}

	var total float64
	for _, v := range vqs {
		total += v.Weight
	}

	r := rng.Float64() * total
	for _, v := range vqs {
		r -= v.Weight
		if r <= 0 {
			return v.VQ
		}
	}
	return vqs[len(vqs)-1].VQ
}
