//go:build postgres

package evidence

import (
	"context"
	"sort"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

func (store *PostgresDistributionStore) GetResponseWorkspace(ctx context.Context, session DistributionAccessSession, request Request, now time.Time) (ResponseWorkspaceView, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	defer tx.Rollback(ctx)
	state, err := loadPostgresWorkspaceState(ctx, tx, session, request, now, false)
	if err != nil {
		return ResponseWorkspaceView{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	return state.View, nil
}

func (store *PostgresDistributionStore) SaveResponseWorkspace(ctx context.Context, command workspaceSaveCommand) (ResponseWorkspaceView, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	defer tx.Rollback(ctx)
	state, err := loadPostgresWorkspaceState(ctx, tx, command.Session, command.Request, command.Now, true)
	if err != nil {
		return ResponseWorkspaceView{}, err
	}
	if command.Input.ExpectedVersion > state.View.Workspace.Version {
		return ResponseWorkspaceView{}, WorkspaceConflict{CurrentVersion: state.View.Workspace.Version}
	}
	if conflict := postgresWorkspaceConflict(state.View, command.Input.Edits); conflict != nil {
		return ResponseWorkspaceView{}, *conflict
	}
	merged := cloneAnswerValues(state.View.Answers)
	changes := make([]FieldEdit, 0, len(command.Input.Edits))
	for _, edit := range command.Input.Edits {
		if answerValueEqual(merged[edit.FieldID], edit.Value) {
			continue
		}
		applyWorkspaceEdit(merged, edit)
		changes = append(changes, edit)
	}
	if command.Validate == nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	if err := command.Validate(merged); err != nil {
		return ResponseWorkspaceView{}, err
	}
	if len(changes) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
		}
		return state.View, nil
	}
	if err := insertPostgresWorkspaceEdits(ctx, tx, &state.View, command, changes); err != nil {
		return ResponseWorkspaceView{}, err
	}
	if err := updatePostgresWorkspaceRow(ctx, tx, state.View.Workspace); err != nil {
		return ResponseWorkspaceView{}, err
	}
	if err := appendPostgresWorkspaceEvent(ctx, tx, command.Session, state.DistributionVersion, "FORM_RESPONSE_WORKSPACE_SAVED", state.View.Workspace.Version, command.Now); err != nil {
		return ResponseWorkspaceView{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	return state.View, nil
}

func (store *PostgresDistributionStore) ClearResponseWorkspace(ctx context.Context, command workspaceSaveCommand) (ResponseWorkspaceView, error) {
	view, err := store.GetResponseWorkspace(ctx, command.Session, command.Request, command.Now)
	if err != nil {
		return ResponseWorkspaceView{}, err
	}
	if command.Input.ExpectedVersion != view.Workspace.Version {
		return ResponseWorkspaceView{}, WorkspaceConflict{CurrentVersion: view.Workspace.Version}
	}
	if len(view.Answers) == 0 {
		return view, nil
	}
	fieldIDs := make([]string, 0, len(view.Answers))
	for fieldID := range view.Answers {
		fieldIDs = append(fieldIDs, fieldID)
	}
	sort.Strings(fieldIDs)
	command.Input.Edits = make([]FieldEdit, 0, len(fieldIDs))
	for _, fieldID := range fieldIDs {
		command.Input.Edits = append(command.Input.Edits, FieldEdit{FieldID: fieldID, BaseSequence: view.FieldSequences[fieldID]})
	}
	return store.SaveResponseWorkspace(ctx, command)
}

func (store *PostgresDistributionStore) SubmitResponseWorkspace(ctx context.Context, command workspaceSubmitCommand) (WorkspaceSubmissionResult, error) {
	if store == nil || store.repo == nil || store.repo.pool == nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	tx, err := store.repo.pool.Begin(ctx)
	if err != nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	defer tx.Rollback(ctx)
	state, err := loadPostgresWorkspaceState(ctx, tx, command.Session, command.Request, command.Now, true)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	if command.Input.ExpectedVersion != state.View.Workspace.Version {
		return WorkspaceSubmissionResult{}, WorkspaceConflict{CurrentVersion: state.View.Workspace.Version}
	}
	answers := cloneAnswerValues(state.View.Answers)
	if command.Validate == nil || command.BuildRevision == nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	if err := command.Validate(answers); err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	metadata, err := command.BuildRevision(answers)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	submissionID, err := id.NewUUIDv7()
	if err != nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	revisionID, err := id.NewUUIDv7()
	if err != nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	revisionNumber := int64(1)
	if state.View.CurrentRevision != nil {
		revisionNumber = state.View.CurrentRevision.Revision + 1
		metadata.SupersedesRevisionID = state.View.CurrentRevision.ID
	}
	metadata.ID = revisionID
	metadata.TenantID = command.Session.TenantID
	metadata.LegalEntityID = command.Session.LegalEntityID
	metadata.DistributionID = command.Session.DistributionID
	metadata.WorkspaceID = state.View.Workspace.ID
	metadata.SubmissionID = submissionID
	metadata.Revision = revisionNumber
	metadata.Current = true
	metadata.CreatedAt = command.Now.UTC()

	receipt, err := insertPostgresWorkspaceSubmission(ctx, tx, command, state.View, submissionID, answers)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	if err := insertPostgresResponseRevision(ctx, tx, metadata, state.View.CurrentRevision); err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	state.View.Workspace.Version++
	state.View.Workspace.UpdatedAt = command.Now.UTC()
	state.View.CurrentRevision = &metadata
	if err := updatePostgresWorkspaceRow(ctx, tx, state.View.Workspace); err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	if err := appendPostgresWorkspaceEvent(ctx, tx, command.Session, state.DistributionVersion, "FORM_RESPONSE_REVISION_SUBMITTED", revisionNumber, command.Now); err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	return WorkspaceSubmissionResult{Workspace: state.View.Workspace, Revision: metadata, Submission: receipt}, nil
}

func (store *PostgresDistributionStore) ValidateWorkspaceAnswers(ctx context.Context, request Request, answers map[string]formcontract.AnswerValue, requireComplete bool) error {
	if store == nil || store.repo == nil {
		return ErrWorkspaceUnavailable
	}
	validator := NewService(store.repo, nil)
	return validator.validateAnswerSet(ctx, request, answers, requireComplete)
}

func postgresWorkspaceConflict(view ResponseWorkspaceView, edits []FieldEdit) *WorkspaceConflict {
	conflict := WorkspaceConflict{CurrentVersion: view.Workspace.Version}
	for _, edit := range edits {
		if view.FieldSequences[edit.FieldID] == edit.BaseSequence {
			continue
		}
		conflict.Changed = append(conflict.Changed, FieldChange{
			FieldID: edit.FieldID, ServerValue: view.Answers[edit.FieldID], Sequence: view.FieldSequences[edit.FieldID],
		})
	}
	if len(conflict.Changed) == 0 {
		return nil
	}
	sort.Slice(conflict.Changed, func(i, j int) bool { return conflict.Changed[i].FieldID < conflict.Changed[j].FieldID })
	return &conflict
}

var (
	_ responseWorkspaceStore             = (*PostgresDistributionStore)(nil)
	_ workspaceAnswerValidationProvider = (*PostgresDistributionStore)(nil)
)
