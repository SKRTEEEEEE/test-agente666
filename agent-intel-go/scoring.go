package main

import (
	"math"
	"time"
)

// CalculateScore computes a priority score for a task based on 5 metrics
// Returns a value between 0 and 1, where higher = higher priority
func CalculateScore(task *TaskMetrics) float64 {
	// Calculate normalized scores for each metric (0-1 range)
	ageScore := normalizeAge(task.CreatedAt)
	activityScore := normalizeRecentActivity(task.LastSuccessAt)
	runtimeScore := normalizeRuntime(task.AvgRuntimeMs)
	loadScore := normalizeLoad(task.PendingTasksCount)
	sizeScore := normalizeSize(task.SizeBytes)

	// Weighted combination
	totalScore := (ageScore * weightAge) +
		(activityScore * weightRecentActivity) +
		(runtimeScore * weightRuntime) +
		(loadScore * weightLoad) +
		(sizeScore * weightSize)

	// Ensure score is within bounds
	if totalScore < 0 {
		totalScore = 0
	}
	if totalScore > 1 {
		totalScore = 1
	}

	return totalScore
}

// normalizeAge converts task age to a 0-1 score
// Older tasks get higher scores (more urgent)
func normalizeAge(createdAt time.Time) float64 {
	if createdAt.IsZero() {
		return 0.5 // Default for unknown age
	}

	ageHours := time.Since(createdAt).Hours()

	// Linear interpolation: 0 hours = 0.0, maxAgeHours = 1.0
	normalized := ageHours / maxAgeHours

	// Cap at 1.0
	if normalized > 1.0 {
		normalized = 1.0
	}

	return normalized
}

// normalizeRecentActivity converts last success time to a 0-1 score
// More recent activity gets higher scores
func normalizeRecentActivity(lastSuccess time.Time) float64 {
	if lastSuccess.IsZero() {
		return 0.0 // Never succeeded = lowest priority
	}

	hoursSince := time.Since(lastSuccess).Hours()

	// Inverse: recent = high score, old = low score
	normalized := 1.0 - (hoursSince / maxRecentHours)

	// Ensure bounds
	if normalized < 0 {
		normalized = 0
	}
	if normalized > 1 {
		normalized = 1
	}

	return normalized
}

// normalizeRuntime converts average runtime to a 0-1 score
// Faster tasks get higher scores
func normalizeRuntime(runtimeMs int64) float64 {
	if runtimeMs <= 0 {
		// Unknown runtime, return middle score
		return 0.5
	}

	// Inverse: shorter runtime = higher score
	normalized := 1.0 - (float64(runtimeMs) / float64(maxRuntimeMs))

	// Ensure bounds
	if normalized < 0 {
		normalized = 0
	}
	if normalized > 1 {
		normalized = 1
	}

	return normalized
}

// normalizeLoad converts pending task count to a 0-1 score
// Fewer pending tasks = higher score (less busy repo)
func normalizeLoad(pendingCount int) float64 {
	if pendingCount < 0 {
		pendingCount = 0
	}

	// Inverse: less load = higher score
	normalized := 1.0 - (float64(pendingCount) / float64(maxPendingTasks))

	// Ensure bounds
	if normalized < 0 {
		normalized = 0
	}
	if normalized > 1 {
		normalized = 1
	}

	return normalized
}

// normalizeSize converts task file size to a 0-1 score
// Smaller tasks get higher scores
func normalizeSize(sizeBytes int64) float64 {
	if sizeBytes <= 0 {
		// Unknown size, return middle score
		return 0.5
	}

	// Inverse: smaller size = higher score
	normalized := 1.0 - (float64(sizeBytes) / float64(maxSizeBytes))

	// Ensure bounds
	if normalized < 0 {
		normalized = 0
	}
	if normalized > 1 {
		normalized = 1
	}

	return normalized
}

// GetScoreExplanation returns a human-readable explanation of the score
func GetScoreExplanation(task *TaskMetrics) map[string]interface{} {
	ageScore := normalizeAge(task.CreatedAt)
	activityScore := normalizeRecentActivity(task.LastSuccessAt)
	runtimeScore := normalizeRuntime(task.AvgRuntimeMs)
	loadScore := normalizeLoad(task.PendingTasksCount)
	sizeScore := normalizeSize(task.SizeBytes)

	return map[string]interface{}{
		"total_score": CalculateScore(task),
		"breakdown": map[string]interface{}{
			"age": map[string]interface{}{
				"score":     ageScore,
				"weight":    weightAge,
				"weighted":  ageScore * weightAge,
				"age_hours": math.Round(time.Since(task.CreatedAt).Hours()*100) / 100,
			},
			"recent_activity": map[string]interface{}{
				"score":       activityScore,
				"weight":      weightRecentActivity,
				"weighted":    activityScore * weightRecentActivity,
				"hours_since": math.Round(time.Since(task.LastSuccessAt).Hours()*100) / 100,
			},
			"runtime": map[string]interface{}{
				"score":      runtimeScore,
				"weight":     weightRuntime,
				"weighted":   runtimeScore * weightRuntime,
				"runtime_ms": task.AvgRuntimeMs,
			},
			"load": map[string]interface{}{
				"score":         loadScore,
				"weight":        weightLoad,
				"weighted":      loadScore * weightLoad,
				"pending_count": task.PendingTasksCount,
			},
			"size": map[string]interface{}{
				"score":      sizeScore,
				"weight":     weightSize,
				"weighted":   sizeScore * weightSize,
				"size_bytes": task.SizeBytes,
			},
		},
	}
}
