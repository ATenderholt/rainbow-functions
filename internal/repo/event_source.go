package repo

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/ATenderholt/rainbow-functions/internal/domain"
	"github.com/ATenderholt/rainbow-functions/pkg/database"
	"github.com/google/uuid"
)

type EventSourceRepository struct {
	db database.Database
}

func NewEventSourceRepository(db database.Database) *EventSourceRepository {
	return &EventSourceRepository{db}
}

func (e *EventSourceRepository) InsertEventSource(ctx context.Context, eventSource domain.EventSource) error {
	_, err := e.db.InsertOne(
		ctx,
		`INSERT INTO lambda_event_source (uuid, enabled, arn, function_id, batch_size, last_modified_on)
					VALUES (?, ?, ?, ?, ?, ?)
		`,
		eventSource.UUID.String(),
		eventSource.Enabled,
		eventSource.Arn,
		eventSource.Function.ID,
		eventSource.BatchSize,
		eventSource.LastModified,
	)

	if err != nil {
		e := Error{"unable insert save Event Source " + eventSource.UUID.String(), err}
		logger.Error(e)
		return e
	}

	return nil
}

func (e *EventSourceRepository) GetEventSource(ctx context.Context, id string) (*domain.EventSource, error) {
	logger.Infof("Loading Event Source %s", id)

	var err error
	var eventSource domain.EventSource
	eventSource.UUID, err = uuid.Parse(id)
	if err != nil {
		err := Error{"unable to parse Event Source id " + id, err}
		logger.Error(err)
		return nil, err
	}

	row := e.db.QueryRowContext(
		ctx,
		`SELECT enabled, arn, function_id, batch_size, last_modified_on FROM lambda_event_source WHERE uuid=?`,
		id,
	)

	var functionId int64
	err = row.Scan(
		&eventSource.Enabled,
		&eventSource.Arn,
		&functionId,
		&eventSource.BatchSize,
		&eventSource.LastModified,
	)

	switch {
	case err == sql.ErrNoRows:
		logger.Warnf("Event Source %s not found", id)
		return nil, nil
	case err != nil:
		e := Error{"unable to find Event Source " + id, err}
		logger.Error(e)
		return nil, e
	}

	row = e.db.QueryRowContext(
		ctx,
		`SELECT name, version FROM lambda_function WHERE id=? ORDER BY version DESC LIMIT 1`,
		functionId,
	)

	var function domain.Function
	err = row.Scan(
		&function.FunctionName,
		&function.Version,
	)

	if err != nil {
		msg := fmt.Sprintf("Unable to find Function %d for Event Source %s", functionId, id)
		e := Error{msg, err}
		logger.Error(e)
		return nil, e
	}

	eventSource.Function = &function

	return &eventSource, nil
}

func (e *EventSourceRepository) GetAllEventSources(ctx context.Context) ([]domain.EventSource, error) {
	logger.Info("Getting all Event Sources")

	var results []domain.EventSource
	rows, err := e.db.QueryContext(
		ctx,
		`SELECT uuid, enabled, arn, function_id, batch_size, last_modified_on FROM lambda_event_source`,
	)

	switch {
	case err == sql.ErrNoRows:
		logger.Warn("Unable to find any event sources.")
		return results, nil
	case err != nil:
		e := Error{"Unable to find Event Sources", err}
		logger.Error(e)
		return nil, e
	}

	stmt, err := e.db.PrepareContext(ctx, `SELECT name, version FROM lambda_function WHERE id=? ORDER BY version DESC LIMIT 1`)
	if err != nil {
		e := Error{"Unable to prepare statement for GetAllEventSources", err}
		logger.Error(e)
		return nil, e
	}
	defer stmt.Close()

	for rows.Next() {
		var eventSource domain.EventSource
		var functionId int64
		err = rows.Scan(
			&eventSource.UUID,
			&eventSource.Enabled,
			&eventSource.Arn,
			&functionId,
			&eventSource.BatchSize,
			&eventSource.LastModified,
		)

		if err != nil {
			e := RowError{
				Op:   "GetAllEventSources",
				Row:  len(results),
				Base: err,
			}
			logger.Error(e)
			return nil, e
		}

		row := stmt.QueryRowContext(ctx, functionId)
		var function domain.Function
		err = row.Scan(
			&function.FunctionName,
			&function.Version,
		)

		if err != nil {
			e := RowError{
				Op:   "GetAllEventSources HydrateFunction",
				Row:  len(results),
				Base: err,
			}
			logger.Error(e)
			return nil, e
		}

		eventSource.Function = &function

		results = append(results, eventSource)
	}

	return results, nil
}
