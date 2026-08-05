//go:build postgres

package continuity

func (r *PostgresRepository) ProjectionQueuedWithCommands() bool { return true }

var _ TransactionalProjectionRepository = (*PostgresRepository)(nil)
