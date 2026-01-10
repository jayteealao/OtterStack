# Progress Callback API Design

**Status:** Design Proposal (Not Implemented)
**Created:** 2026-01-10
**Type:** Enhancement
**Complexity:** Low
**Risk:** Low
**Priority:** P3

---

## Overview

Design a structured progress callback API for OtterStack deployments to enable real-time progress updates for UI integrations (web dashboards, CLIs, monitoring tools). This design document outlines the API structure, usage patterns, and implementation approaches without requiring immediate implementation.

**Important:** This is an optional enhancement. Do NOT implement unless users specifically request real-time progress updates for UI integration.

## Problem Statement

### Current State

OtterStack currently uses simple callback functions for deployment status:

```go
type DeployOptions struct {
    // ... other fields
    OnStatus  func(msg string) // Basic status messages
    OnVerbose func(msg string) // Verbose/debug messages
}
```

**Limitations:**
- Unstructured text messages only
- No progress percentage or phase information
- No machine-readable status
- Difficult to build UIs that show progress bars or deployment phases
- Cannot distinguish between error, warning, and info messages
- No timing information for progress tracking

### Use Cases

1. **Web Dashboard:** Display deployment progress with phase indicators and progress bars
2. **CLI with Progress Bars:** Show interactive deployment progress (e.g., using tview, bubbletea)
3. **Monitoring Integration:** Export deployment metrics to Prometheus, Datadog, etc.
4. **Webhook Notifications:** Send structured deployment events to external systems
5. **Audit Logging:** Capture detailed deployment timeline with phase durations

## Proposed Solution

### Design Goals

1. **Backward Compatible:** Existing OnStatus/OnVerbose callbacks continue to work
2. **Structured Data:** Provide machine-readable progress information
3. **Optional:** Progress callbacks are opt-in; deployments work without them
4. **Simple:** Easy to use for common cases (CLI, web UI)
5. **Extensible:** Support custom progress tracking needs

### Core Data Structures

```go
// Package orchestrator

// ProgressPhase represents the current deployment phase
type ProgressPhase string

const (
    PhaseInitializing  ProgressPhase = "initializing"  // Acquiring locks, validating
    PhaseFetching      ProgressPhase = "fetching"      // Git fetch
    PhaseResolving     ProgressPhase = "resolving"     // Resolving git ref
    PhaseWorktree      ProgressPhase = "worktree"      // Creating worktree
    PhaseValidating    ProgressPhase = "validating"    // Compose file validation
    PhasePulling       ProgressPhase = "pulling"       // Image pull
    PhaseStarting      ProgressPhase = "starting"      // docker compose up
    PhaseHealthCheck   ProgressPhase = "health_check"  // Waiting for health
    PhaseTraefikLabels ProgressPhase = "traefik"       // Applying Traefik labels
    PhaseCleanup       ProgressPhase = "cleanup"       // Stopping old containers
    PhaseComplete      ProgressPhase = "complete"      // Deployment finished
    PhaseFailed        ProgressPhase = "failed"        // Deployment failed
)

// ProgressLevel indicates the message severity
type ProgressLevel string

const (
    LevelInfo    ProgressLevel = "info"    // Informational messages
    LevelVerbose ProgressLevel = "verbose" // Debug/verbose messages
    LevelWarning ProgressLevel = "warning" // Non-fatal warnings
    LevelError   ProgressLevel = "error"   // Errors (deployment may fail)
    LevelSuccess ProgressLevel = "success" // Success messages
)

// ProgressUpdate contains structured deployment progress information
type ProgressUpdate struct {
    // Deployment identification
    ProjectName   string        `json:"project_name"`
    DeploymentID  int64         `json:"deployment_id,omitempty"`

    // Progress tracking
    Phase         ProgressPhase `json:"phase"`
    PhaseProgress float64       `json:"phase_progress"` // 0.0 to 1.0 for current phase
    TotalProgress float64       `json:"total_progress"` // 0.0 to 1.0 overall

    // Message details
    Level         ProgressLevel `json:"level"`
    Message       string        `json:"message"`

    // Timing
    Timestamp     time.Time     `json:"timestamp"`
    ElapsedTime   time.Duration `json:"elapsed_time,omitempty"`

    // Optional context
    Metadata      map[string]interface{} `json:"metadata,omitempty"`
    Error         error                  `json:"-"` // Not serialized to JSON
}

// ProgressCallback is called with deployment progress updates
type ProgressCallback func(update ProgressUpdate)

// DeployOptions with progress callback support
type DeployOptions struct {
    GitRef       string
    Timeout      time.Duration
    SkipPull     bool
    DataDir      string

    // Legacy callbacks (still supported)
    OnStatus     func(msg string)
    OnVerbose    func(msg string)

    // New structured callback (optional)
    OnProgress   ProgressCallback
}
```

### Implementation Approach

#### Option 1: Wrapper Functions (Recommended)

Minimal changes to existing code. Add helper functions that emit both legacy and structured callbacks:

```go
// deployer.go

type progressTracker struct {
    projectName  string
    deploymentID int64
    startTime    time.Time
    currentPhase ProgressPhase
    onProgress   ProgressCallback
    onStatus     func(string)
    onVerbose    func(string)
}

func (p *progressTracker) emit(level ProgressLevel, phase ProgressPhase, msg string, metadata map[string]interface{}) {
    // Always call legacy callbacks
    switch level {
    case LevelInfo, LevelSuccess:
        if p.onStatus != nil {
            p.onStatus(msg)
        }
    case LevelVerbose:
        if p.onVerbose != nil {
            p.onVerbose(msg)
        }
    }

    // Call structured callback if provided
    if p.onProgress != nil {
        update := ProgressUpdate{
            ProjectName:  p.projectName,
            DeploymentID: p.deploymentID,
            Phase:        phase,
            Level:        level,
            Message:      msg,
            Timestamp:    time.Now(),
            ElapsedTime:  time.Since(p.startTime),
            Metadata:     metadata,
        }

        // Calculate progress based on phase
        update.TotalProgress = p.calculateProgress(phase)

        p.onProgress(update)
    }
}

func (p *progressTracker) calculateProgress(phase ProgressPhase) float64 {
    // Simple linear progress based on phases
    phaseWeights := map[ProgressPhase]float64{
        PhaseInitializing:  0.05,
        PhaseFetching:      0.10,
        PhaseResolving:     0.15,
        PhaseWorktree:      0.20,
        PhaseValidating:    0.25,
        PhasePulling:       0.40,
        PhaseStarting:      0.60,
        PhaseHealthCheck:   0.80,
        PhaseTraefikLabels: 0.90,
        PhaseCleanup:       0.95,
        PhaseComplete:      1.00,
    }
    return phaseWeights[phase]
}

// Example usage in Deploy()
func (d *Deployer) Deploy(ctx context.Context, project *state.Project, opts DeployOptions) (*DeployResult, error) {
    // Initialize progress tracker
    tracker := &progressTracker{
        projectName: project.Name,
        startTime:   time.Now(),
        onProgress:  opts.OnProgress,
        onStatus:    opts.OnStatus,
        onVerbose:   opts.OnVerbose,
    }

    // Emit progress updates throughout deployment
    tracker.emit(LevelInfo, PhaseInitializing, "Acquiring deployment lock...", nil)

    // ... existing code ...

    tracker.emit(LevelInfo, PhaseFetching, "Fetching latest changes...", nil)
    if err := d.gitMgr.Fetch(ctx); err != nil {
        tracker.emit(LevelError, PhaseFetching, "Failed to fetch", map[string]interface{}{
            "error": err.Error(),
        })
        return nil, err
    }

    // ... more progress emissions ...
}
```

#### Option 2: JSON Output Mode

Alternative approach using structured JSON output directly to stdout (similar to Docker CLI's `--format json`):

```go
type DeployOptions struct {
    // ... existing fields ...
    OutputFormat string // "text" (default), "json"
}

// In Deploy(), emit JSON for each status update
if opts.OutputFormat == "json" {
    json.NewEncoder(os.Stdout).Encode(ProgressUpdate{
        Phase:   PhaseFetching,
        Message: "Fetching latest changes...",
        // ... other fields
    })
}
```

**Pros:**
- No callback handling needed
- Works with any language (parse JSON stdout)
- Easy to pipe to jq, logging systems, etc.

**Cons:**
- Mixes progress with Docker output
- Harder to use in-process (library usage)
- Requires stdout parsing

### Example Usage Patterns

#### 1. CLI with Progress Bar

```go
package main

import (
    "fmt"
    "github.com/charmbracelet/bubbles/progress"
    tea "github.com/charmbracelet/bubbletea"
)

type model struct {
    progress progress.Model
    phase    string
    message  string
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ProgressUpdate:
        m.progress.SetPercent(msg.TotalProgress)
        m.phase = string(msg.Phase)
        m.message = msg.Message
    }
    return m, nil
}

// Deploy with callback
deployer.Deploy(ctx, project, DeployOptions{
    OnProgress: func(update ProgressUpdate) {
        program.Send(update) // Send to bubbletea
    },
})
```

#### 2. Web Dashboard (WebSocket)

```go
// Server-side
func handleDeploy(w http.ResponseWriter, r *http.Request) {
    conn, _ := upgrader.Upgrade(w, r, nil)

    deployer.Deploy(ctx, project, DeployOptions{
        OnProgress: func(update ProgressUpdate) {
            // Send JSON to WebSocket
            conn.WriteJSON(update)
        },
    })
}

// Client-side JavaScript
const ws = new WebSocket('ws://localhost:8080/deploy');
ws.onmessage = (event) => {
    const update = JSON.parse(event.data);
    updateProgressBar(update.total_progress * 100);
    updatePhaseText(update.phase);
    updateLog(update.message);
};
```

#### 3. Prometheus Metrics Export

```go
import "github.com/prometheus/client_golang/prometheus"

var deploymentPhase = prometheus.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "otterstack_deployment_phase",
        Help: "Current deployment phase",
    },
    []string{"project", "phase"},
)

deployer.Deploy(ctx, project, DeployOptions{
    OnProgress: func(update ProgressUpdate) {
        deploymentPhase.WithLabelValues(
            update.ProjectName,
            string(update.Phase),
        ).Set(update.TotalProgress)
    },
})
```

#### 4. Simple CLI (Backward Compatible)

```go
// Existing code continues to work
deployer.Deploy(ctx, project, DeployOptions{
    OnStatus: func(msg string) {
        fmt.Println(msg)
    },
})
```

## Evaluation of Approaches

### Text Parsing vs Structured Output

| Aspect | Current (Text) | Structured (Callbacks) | JSON Output |
|--------|---------------|----------------------|-------------|
| **Ease of Use** | Simple | Moderate | Complex |
| **Machine Readable** | No | Yes | Yes |
| **In-Process** | Yes | Yes | No |
| **Language Agnostic** | No | No | Yes |
| **Backward Compatible** | N/A | Yes | Partial |
| **Progress Tracking** | Hard | Easy | Easy |
| **UI Integration** | Hard | Easy | Moderate |

**Recommendation:** Use structured callbacks (Option 1) with optional JSON output mode for external integrations.

## Phase Transition Matrix

Visual guide for implementing progress tracking:

```
┌─────────────────┬──────────────────────────────────────────────────────┐
│ Phase           │ Triggers / Duration Estimate                         │
├─────────────────┼──────────────────────────────────────────────────────┤
│ Initializing    │ Lock acquisition (< 1s)                             │
│ Fetching        │ Git fetch (2-10s, depends on network)               │
│ Resolving       │ Git ref resolution (< 1s)                           │
│ Worktree        │ Worktree creation (1-5s, depends on repo size)      │
│ Validating      │ Compose validation (< 1s)                           │
│ Pulling         │ Image pull (10-120s, highly variable)               │
│ Starting        │ docker compose up (5-30s)                           │
│ Health Check    │ Container health checks (0-300s, configurable)      │
│ Traefik Labels  │ Override file apply (2-5s)                          │
│ Cleanup         │ Stop old containers (5-10s)                         │
│ Complete        │ Final status update (< 1s)                          │
└─────────────────┴──────────────────────────────────────────────────────┘
```

## Implementation Checklist (When Requested)

**DO NOT IMPLEMENT** unless users request this feature. If implemented:

- [ ] Add ProgressUpdate struct to `internal/orchestrator/deployer.go`
- [ ] Add ProgressCallback type and OnProgress to DeployOptions
- [ ] Create progressTracker helper struct
- [ ] Update Deploy() to emit progress at each phase transition
- [ ] Add progress calculation logic
- [ ] Write unit tests for progress tracking
- [ ] Add example CLI implementation using progress bars
- [ ] Update documentation with usage examples
- [ ] Consider adding `--format json` flag for JSON output mode

## Alternative Considerations

### 1. Event-Driven Architecture

Instead of callbacks, use Go channels:

```go
type DeployOptions struct {
    ProgressChan chan<- ProgressUpdate
}

// Caller receives on channel
progressChan := make(chan ProgressUpdate, 10)
go func() {
    for update := range progressChan {
        fmt.Printf("Progress: %.0f%%\n", update.TotalProgress * 100)
    }
}()

deployer.Deploy(ctx, project, DeployOptions{
    ProgressChan: progressChan,
})
```

**Pros:** More idiomatic Go, better backpressure handling
**Cons:** More complex for simple use cases, requires goroutine management

### 2. Observer Pattern

Register multiple observers:

```go
deployer.RegisterObserver(progressBarObserver)
deployer.RegisterObserver(loggingObserver)
deployer.Deploy(ctx, project, opts)
```

**Pros:** Multiple consumers, decoupled
**Cons:** More boilerplate, unnecessary for most cases

## Open Questions

1. **Granular Image Pull Progress?** Should we parse Docker output to show per-image pull progress?
   - **Answer:** Not initially. Docker output streaming is sufficient. Can add later if needed.

2. **Cancellation via Progress Callback?** Should callbacks be able to cancel deployment?
   - **Answer:** No. Use context cancellation. Progress is read-only.

3. **Historical Progress?** Should we store progress updates in the database?
   - **Answer:** Not initially. Focus on real-time updates. Can add audit logging later.

4. **Progress Estimates?** Should we estimate time remaining based on historical data?
   - **Answer:** Out of scope. Phase weights are sufficient.

## Migration Path

### Phase 1: Add Structures (No Breaking Changes)
- Add ProgressUpdate struct
- Add OnProgress callback (optional)
- Keep existing OnStatus/OnVerbose
- All existing code works unchanged

### Phase 2: Adopt Internally
- Update CLI commands to use OnProgress
- Add progress bar to interactive deployments
- Keep text output for non-interactive

### Phase 3: External Integration
- Document API for web dashboards
- Add examples for WebSocket integration
- Consider adding gRPC/REST API wrapper

## Related Work

### Similar Systems

1. **Docker CLI Progress:** `docker pull` shows layer-by-layer progress
2. **Kubernetes Events:** Structured event stream for pod lifecycle
3. **Terraform:** Detailed phase output with resource counts
4. **Ansible:** Task-by-task progress with timing

### Lessons Learned

- **Docker:** Good visual progress, but hard to parse programmatically
- **Kubernetes:** Excellent structured events, but complex API
- **Balance:** Provide both human-friendly text and machine-readable data

## Conclusion

This design provides a structured, backward-compatible approach to deployment progress tracking. The callback-based approach (Option 1) is recommended for its simplicity and flexibility.

**Key Takeaways:**
1. Optional enhancement - only implement if users request it
2. Backward compatible with existing OnStatus/OnVerbose callbacks
3. Structured data enables rich UI integrations
4. Simple to use for common cases (CLI, web dashboard)
5. Extensible for advanced use cases (metrics, logging)

**Next Steps (Only if Requested):**
1. Gather user feedback on whether this is needed
2. Validate use cases with actual UI integration attempts
3. Implement minimal viable version (ProgressUpdate + basic callbacks)
4. Iterate based on real-world usage

---

**Document Version:** 1.0
**Last Updated:** 2026-01-10
**Reviewers:** (pending)
