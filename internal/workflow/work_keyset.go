package workflow

// workFollows returns the next position in the active-work ordering shared
// by the two source reads and the merged actor queue.
func workFollows(task Task, after WorkPosition) bool {
	if after.DueAt != nil {
		if task.DueAt == nil {
			return true
		}
		if task.DueAt.Before(*after.DueAt) {
			return false
		}
		if task.DueAt.After(*after.DueAt) {
			return true
		}
	} else if task.DueAt != nil {
		return false
	}
	if !task.UpdatedAt.Equal(after.UpdatedAt) {
		return task.UpdatedAt.Before(after.UpdatedAt)
	}
	return task.ID > after.ID
}

func positionForTask(task Task) WorkPosition {
	return WorkPosition{DueAt: task.DueAt, UpdatedAt: task.UpdatedAt, ID: task.ID}
}
