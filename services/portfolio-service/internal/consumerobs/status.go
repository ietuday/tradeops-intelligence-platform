package consumerobs

import "time"

func CalculateLag(latestOffset, currentOffset int64) int64 {
	if latestOffset <= currentOffset {
		return 0
	}
	if currentOffset < 0 {
		currentOffset = 0
	}
	return latestOffset - currentOffset
}

func ClassifyConsumer(totalLag int64, consecutiveErrors int, lastProcessed *time.Time, now time.Time, cfg Config) string {
	if totalLag >= cfg.LagCriticalThreshold && cfg.LagCriticalThreshold > 0 {
		return StatusDegraded
	}
	if lastProcessed == nil && totalLag > 0 {
		return StatusUnknown
	}
	if lastProcessed != nil && cfg.StalledAfter > 0 && totalLag > 0 && now.Sub(lastProcessed.UTC()) >= cfg.StalledAfter {
		return StatusStalled
	}
	if totalLag >= cfg.LagWarnThreshold && cfg.LagWarnThreshold > 0 {
		return StatusDegraded
	}
	if consecutiveErrors >= 3 {
		return StatusDegraded
	}
	return StatusHealthy
}

func ClassifyDLQ(messageCount int64, oldestAge time.Duration, cfg Config) string {
	if messageCount <= 0 {
		return StatusHealthy
	}
	if cfg.DLQOldestAgeCritical > 0 && oldestAge >= cfg.DLQOldestAgeCritical {
		return "critical"
	}
	return StatusDegraded
}

func AggregateStatus(consumers []ConsumerStatus, dlq []DLQStatus) string {
	overall := StatusHealthy
	for _, status := range consumers {
		if status.Status == StatusStalled {
			return StatusStalled
		}
		if status.Status == StatusDegraded || status.Status == StatusUnknown {
			overall = StatusDegraded
		}
	}
	for _, status := range dlq {
		if status.Status == "critical" {
			return StatusDegraded
		}
		if status.Status == StatusDegraded {
			overall = StatusDegraded
		}
	}
	return overall
}
