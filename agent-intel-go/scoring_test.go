package main

import (
	"math"
	"testing"
	"time"
)

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name     string
		task     *TaskMetrics
		expected float64
		delta    float64 // Allow for small floating-point differences
	}{
		{
			name: "high priority old task",
			task: &TaskMetrics{
				TaskID:            "task-1",
				CreatedAt:         time.Now().Add(-48 * time.Hour), // 2 days old
				LastSuccessAt:     time.Now().Add(-1 * time.Hour),  // Recent success
				AvgRuntimeMs:      5000,                            // 5 seconds
				PendingTasksCount: 1,                               // Low load
				SizeBytes:         1024,                            // 1KB
				PipelineRuntimeMs: 0,
				AssignedAt:        time.Time{},
				Status:            StatusPending,
			},
			expected: 0.75, // High score due to age and low runtime
			delta:    0.15,
		},
		{
			name: "low priority new task with high load",
			task: &TaskMetrics{
				TaskID:            "task-2",
				CreatedAt:         time.Now().Add(-1 * time.Hour),  // 1 hour old
				LastSuccessAt:     time.Now().Add(-48 * time.Hour), // Old success
				AvgRuntimeMs:      3600000,                         // 1 hour
				PendingTasksCount: 10,                              // High load
				SizeBytes:         1024000,                         // 1MB
				PipelineRuntimeMs: 0,
				AssignedAt:        time.Time{},
				Status:            StatusPending,
			},
			expected: 0.30, // Low score due to recent creation and high load
			delta:    0.15,
		},
		{
			name: "medium priority task",
			task: &TaskMetrics{
				TaskID:            "task-3",
				CreatedAt:         time.Now().Add(-24 * time.Hour), // 1 day old
				LastSuccessAt:     time.Now().Add(-24 * time.Hour), // 1 day ago
				AvgRuntimeMs:      60000,                           // 1 minute
				PendingTasksCount: 5,                               // Medium load
				SizeBytes:         51200,                           // 50KB
				PipelineRuntimeMs: 0,
				AssignedAt:        time.Time{},
				Status:            StatusPending,
			},
			expected: 0.50, // Medium score
			delta:    0.15,
		},
		{
			name: "task with zero metrics",
			task: &TaskMetrics{
				TaskID:            "task-4",
				CreatedAt:         time.Now(),
				LastSuccessAt:     time.Time{}, // Never succeeded
				AvgRuntimeMs:      0,
				PendingTasksCount: 0,
				SizeBytes:         0,
				PipelineRuntimeMs: 0,
				AssignedAt:        time.Time{},
				Status:            StatusPending,
			},
			expected: 0.35, // Base score with minimal data
			delta:    0.15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := CalculateScore(tt.task)

			// Check if score is within expected range
			if math.Abs(score-tt.expected) > tt.delta {
				t.Errorf("CalculateScore() = %v, want %v ± %v", score, tt.expected, tt.delta)
			}

			// Ensure score is between 0 and 1
			if score < 0 || score > 1 {
				t.Errorf("CalculateScore() = %v, must be between 0 and 1", score)
			}
		})
	}
}

func TestNormalizeAge(t *testing.T) {
	tests := []struct {
		name        string
		createdAt   time.Time
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "very old task (7 days)",
			createdAt:   time.Now().Add(-7 * 24 * time.Hour),
			expectedMin: 0.9,
			expectedMax: 1.0,
		},
		{
			name:        "medium age task (12 hours)",
			createdAt:   time.Now().Add(-12 * time.Hour),
			expectedMin: 0.05,
			expectedMax: 0.10,
		},
		{
			name:        "new task (1 hour)",
			createdAt:   time.Now().Add(-1 * time.Hour),
			expectedMin: 0.0,
			expectedMax: 0.2,
		},
		{
			name:        "just created",
			createdAt:   time.Now(),
			expectedMin: 0.0,
			expectedMax: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := normalizeAge(tt.createdAt)

			if normalized < tt.expectedMin || normalized > tt.expectedMax {
				t.Errorf("normalizeAge() = %v, want between %v and %v",
					normalized, tt.expectedMin, tt.expectedMax)
			}

			if normalized < 0 || normalized > 1 {
				t.Errorf("normalizeAge() = %v, must be between 0 and 1", normalized)
			}
		})
	}
}

func TestNormalizeRecentActivity(t *testing.T) {
	tests := []struct {
		name        string
		lastSuccess time.Time
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "very recent success (1 hour)",
			lastSuccess: time.Now().Add(-1 * time.Hour),
			expectedMin: 0.9,
			expectedMax: 1.0,
		},
		{
			name:        "medium recent success (12 hours)",
			lastSuccess: time.Now().Add(-12 * time.Hour),
			expectedMin: 0.90,
			expectedMax: 0.95,
		},
		{
			name:        "old success (7 days)",
			lastSuccess: time.Now().Add(-7 * 24 * time.Hour),
			expectedMin: 0.0,
			expectedMax: 0.2,
		},
		{
			name:        "never succeeded",
			lastSuccess: time.Time{},
			expectedMin: 0.0,
			expectedMax: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := normalizeRecentActivity(tt.lastSuccess)

			if normalized < tt.expectedMin || normalized > tt.expectedMax {
				t.Errorf("normalizeRecentActivity() = %v, want between %v and %v",
					normalized, tt.expectedMin, tt.expectedMax)
			}

			if normalized < 0 || normalized > 1 {
				t.Errorf("normalizeRecentActivity() = %v, must be between 0 and 1", normalized)
			}
		})
	}
}

func TestNormalizeRuntime(t *testing.T) {
	tests := []struct {
		name        string
		runtimeMs   int64
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "very fast (5 seconds)",
			runtimeMs:   5000,
			expectedMin: 0.9,
			expectedMax: 1.0,
		},
		{
			name:        "medium speed (1 minute)",
			runtimeMs:   60000,
			expectedMin: 0.97,
			expectedMax: 0.98,
		},
		{
			name:        "slow (10 minutes)",
			runtimeMs:   600000,
			expectedMin: 0.70,
			expectedMax: 0.80,
		},
		{
			name:        "very slow (40 minutes)",
			runtimeMs:   2400000,
			expectedMin: 0.0,
			expectedMax: 0.2,
		},
		{
			name:        "unknown runtime (0)",
			runtimeMs:   0,
			expectedMin: 0.4,
			expectedMax: 0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := normalizeRuntime(tt.runtimeMs)

			if normalized < tt.expectedMin || normalized > tt.expectedMax {
				t.Errorf("normalizeRuntime() = %v, want between %v and %v",
					normalized, tt.expectedMin, tt.expectedMax)
			}

			if normalized < 0 || normalized > 1 {
				t.Errorf("normalizeRuntime() = %v, must be between 0 and 1", normalized)
			}
		})
	}
}

func TestNormalizeLoad(t *testing.T) {
	tests := []struct {
		name         string
		pendingCount int
		expectedMin  float64
		expectedMax  float64
	}{
		{
			name:         "no load",
			pendingCount: 0,
			expectedMin:  0.9,
			expectedMax:  1.0,
		},
		{
			name:         "low load (2 tasks)",
			pendingCount: 2,
			expectedMin:  0.7,
			expectedMax:  0.9,
		},
		{
			name:         "medium load (5 tasks)",
			pendingCount: 5,
			expectedMin:  0.4,
			expectedMax:  0.6,
		},
		{
			name:         "high load (10 tasks)",
			pendingCount: 10,
			expectedMin:  0.0,
			expectedMax:  0.2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := normalizeLoad(tt.pendingCount)

			if normalized < tt.expectedMin || normalized > tt.expectedMax {
				t.Errorf("normalizeLoad() = %v, want between %v and %v",
					normalized, tt.expectedMin, tt.expectedMax)
			}

			if normalized < 0 || normalized > 1 {
				t.Errorf("normalizeLoad() = %v, must be between 0 and 1", normalized)
			}
		})
	}
}

func TestNormalizeSize(t *testing.T) {
	tests := []struct {
		name        string
		sizeBytes   int64
		expectedMin float64
		expectedMax float64
	}{
		{
			name:        "very small (1KB)",
			sizeBytes:   1024,
			expectedMin: 0.9,
			expectedMax: 1.0,
		},
		{
			name:        "small (10KB)",
			sizeBytes:   10240,
			expectedMin: 0.98,
			expectedMax: 1.0,
		},
		{
			name:        "medium (100KB)",
			sizeBytes:   102400,
			expectedMin: 0.85,
			expectedMax: 0.95,
		},
		{
			name:        "large (1MB)",
			sizeBytes:   1048576,
			expectedMin: 0.0,
			expectedMax: 0.2,
		},
		{
			name:        "unknown size (0)",
			sizeBytes:   0,
			expectedMin: 0.4,
			expectedMax: 0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := normalizeSize(tt.sizeBytes)

			if normalized < tt.expectedMin || normalized > tt.expectedMax {
				t.Errorf("normalizeSize() = %v, want between %v and %v",
					normalized, tt.expectedMin, tt.expectedMax)
			}

			if normalized < 0 || normalized > 1 {
				t.Errorf("normalizeSize() = %v, must be between 0 and 1", normalized)
			}
		})
	}
}

func TestScoreWeights(t *testing.T) {
	// Verify that all weights sum to 1.0
	totalWeight := weightAge + weightRecentActivity + weightRuntime + weightLoad + weightSize

	if math.Abs(totalWeight-1.0) > 0.001 {
		t.Errorf("Score weights sum to %v, must sum to 1.0", totalWeight)
	}

	// Verify individual weights are reasonable
	weights := map[string]float64{
		"age":             weightAge,
		"recent_activity": weightRecentActivity,
		"runtime":         weightRuntime,
		"load":            weightLoad,
		"size":            weightSize,
	}

	for name, weight := range weights {
		if weight < 0 || weight > 1 {
			t.Errorf("Weight %s = %v, must be between 0 and 1", name, weight)
		}
	}
}
