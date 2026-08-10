package scimapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) AuthenticateSource(ctx context.Context, tokenHash []byte) (Source, error) {
	if len(tokenHash) != sha256Size {
		return Source{}, ErrNotFound
	}
	var source Source
	err := r.pool.QueryRow(ctx, `
		SELECT id::text,tenant_id::text,COALESCE(identity_issuer,''),subject_attribute
		FROM scim_sources
		WHERE token_hash=$1 AND status='ACTIVE'`, tokenHash).
		Scan(&source.ID, &source.TenantID, &source.IdentityIssuer, &source.SubjectAttribute)
	if errors.Is(err, pgx.ErrNoRows) {
		return Source{}, ErrNotFound
	}
	if err != nil {
		return Source{}, fmt.Errorf("authenticate SCIM source: %w", err)
	}
	return source, nil
}

const sha256Size = 32

func (r *PostgresRepository) CreateUser(ctx context.Context, source Source, input User) (User, error) {
	input, err := normalizeUser(source, input)
	if err != nil {
		return User{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)

	var existing User
	var deletedAt *time.Time
	query := `
		SELECT su.id::text,su.principal_id::text,COALESCE(su.external_id,''),su.user_name,p.display_name,su.active,su.created_at,su.updated_at,su.deleted_at
		FROM scim_users su
		JOIN principals p ON p.id=su.principal_id AND p.tenant_id=su.tenant_id
		WHERE su.source_id=$1::uuid AND `
	args := []any{source.ID}
	if input.ExternalID != "" {
		query += `su.external_id=$2 ORDER BY su.updated_at DESC LIMIT 1 FOR UPDATE`
		args = append(args, input.ExternalID)
	} else {
		query += `lower(su.user_name)=lower($2) ORDER BY su.updated_at DESC LIMIT 1 FOR UPDATE`
		args = append(args, input.UserName)
	}
	err = tx.QueryRow(ctx, query, args...).Scan(
		&existing.ID, &existing.PrincipalID, &existing.ExternalID, &existing.UserName, &existing.DisplayName,
		&existing.Active, &existing.CreatedAt, &existing.UpdatedAt, &deletedAt,
	)
	if err == nil {
		if deletedAt == nil {
			return User{}, ErrConflict
		}
		if err := ensureUserUnique(ctx, tx, source, existing.ID, input); err != nil {
			return User{}, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE principals SET display_name=$1,status=$2,valid_until=NULL WHERE tenant_id=$3::uuid AND id=$4::uuid`,
			input.DisplayName, principalStatus(input.Active), source.TenantID, existing.PrincipalID); err != nil {
			return User{}, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE scim_users SET external_id=NULLIF($1,''),user_name=$2,active=$3,updated_at=clock_timestamp(),deleted_at=NULL
			WHERE tenant_id=$4::uuid AND source_id=$5::uuid AND id=$6::uuid`,
			input.ExternalID, input.UserName, input.Active, source.TenantID, source.ID, existing.ID); err != nil {
			return User{}, mapPgError(err)
		}
		if err := syncPrincipalIdentity(ctx, tx, source, existing.PrincipalID, input); err != nil {
			return User{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return User{}, err
		}
		return r.GetUser(ctx, source, existing.ID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return User{}, err
	}

	var principalID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO principals(tenant_id,kind,display_name,status,valid_from)
		VALUES($1::uuid,'PERSON',$2,$3,clock_timestamp()) RETURNING id::text`,
		source.TenantID, input.DisplayName, principalStatus(input.Active)).Scan(&principalID); err != nil {
		return User{}, err
	}
	var userID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO scim_users(tenant_id,source_id,principal_id,external_id,user_name,active)
		VALUES($1::uuid,$2::uuid,$3::uuid,NULLIF($4,''),$5,$6) RETURNING id::text`,
		source.TenantID, source.ID, principalID, input.ExternalID, input.UserName, input.Active).Scan(&userID); err != nil {
		return User{}, mapPgError(err)
	}
	if err := syncPrincipalIdentity(ctx, tx, source, principalID, input); err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return r.GetUser(ctx, source, userID)
}

func (r *PostgresRepository) GetUser(ctx context.Context, source Source, id string) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		SELECT su.id::text,su.principal_id::text,COALESCE(su.external_id,''),su.user_name,p.display_name,su.active,su.created_at,su.updated_at
		FROM scim_users su
		JOIN principals p ON p.id=su.principal_id AND p.tenant_id=su.tenant_id
		WHERE su.tenant_id=$1::uuid AND su.source_id=$2::uuid AND su.id::text=$3 AND su.deleted_at IS NULL`,
		source.TenantID, source.ID, strings.TrimSpace(id)).Scan(
		&user.ID, &user.PrincipalID, &user.ExternalID, &user.UserName, &user.DisplayName,
		&user.Active, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (r *PostgresRepository) ListUsers(ctx context.Context, source Source, filter UserFilter, offset, limit int) ([]User, int, error) {
	where := `su.tenant_id=$1::uuid AND su.source_id=$2::uuid AND su.deleted_at IS NULL`
	args := []any{source.TenantID, source.ID}
	if filter.UserName != "" {
		args = append(args, filter.UserName)
		where += fmt.Sprintf(" AND lower(su.user_name)=lower($%d)", len(args))
	}
	if filter.ExternalID != "" {
		args = append(args, filter.ExternalID)
		where += fmt.Sprintf(" AND su.external_id=$%d", len(args))
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM scim_users su WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if limit <= 0 || total == 0 {
		return []User{}, total, nil
	}
	args = append(args, limit, max(offset, 0))
	rows, err := r.pool.Query(ctx, `
		SELECT su.id::text,su.principal_id::text,COALESCE(su.external_id,''),su.user_name,p.display_name,su.active,su.created_at,su.updated_at
		FROM scim_users su JOIN principals p ON p.id=su.principal_id AND p.tenant_id=su.tenant_id
		WHERE `+where+fmt.Sprintf(` ORDER BY lower(su.user_name),su.id LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	users := make([]User, 0, min(limit, total))
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.PrincipalID, &user.ExternalID, &user.UserName, &user.DisplayName, &user.Active, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}
	return users, total, rows.Err()
}

func (r *PostgresRepository) ReplaceUser(ctx context.Context, source Source, id string, input User) (User, error) {
	input, err := normalizeUser(source, input)
	if err != nil {
		return User{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)
	var principalID string
	if err := tx.QueryRow(ctx, `
		SELECT principal_id::text FROM scim_users
		WHERE tenant_id=$1::uuid AND source_id=$2::uuid AND id::text=$3 AND deleted_at IS NULL FOR UPDATE`,
		source.TenantID, source.ID, strings.TrimSpace(id)).Scan(&principalID); errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	} else if err != nil {
		return User{}, err
	}
	if err := ensureUserUnique(ctx, tx, source, id, input); err != nil {
		return User{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE principals SET display_name=$1,status=$2 WHERE tenant_id=$3::uuid AND id=$4::uuid`,
		input.DisplayName, principalStatus(input.Active), source.TenantID, principalID); err != nil {
		return User{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE scim_users SET external_id=NULLIF($1,''),user_name=$2,active=$3,updated_at=clock_timestamp()
		WHERE tenant_id=$4::uuid AND source_id=$5::uuid AND id::text=$6`,
		input.ExternalID, input.UserName, input.Active, source.TenantID, source.ID, id); err != nil {
		return User{}, mapPgError(err)
	}
	if err := syncPrincipalIdentity(ctx, tx, source, principalID, input); err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return r.GetUser(ctx, source, id)
}

func (r *PostgresRepository) DeleteUser(ctx context.Context, source Source, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var principalID string
	if err := tx.QueryRow(ctx, `
		SELECT principal_id::text FROM scim_users
		WHERE tenant_id=$1::uuid AND source_id=$2::uuid AND id::text=$3 AND deleted_at IS NULL FOR UPDATE`,
		source.TenantID, source.ID, strings.TrimSpace(id)).Scan(&principalID); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE scim_users SET active=false,deleted_at=clock_timestamp(),updated_at=clock_timestamp() WHERE tenant_id=$1::uuid AND id::text=$2`, source.TenantID, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE principals SET status='INACTIVE' WHERE tenant_id=$1::uuid AND id=$2::uuid`, source.TenantID, principalID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM directory_group_members WHERE tenant_id=$1::uuid AND scim_user_id::text=$2`, source.TenantID, id); err != nil {
		return err
	}
	if source.IdentityIssuer != "" {
		if _, err := tx.Exec(ctx, `UPDATE principal_identities SET status='REVOKED',updated_at=clock_timestamp() WHERE tenant_id=$1::uuid AND principal_id=$2::uuid AND issuer=$3`, source.TenantID, principalID, source.IdentityIssuer); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) CreateGroup(ctx context.Context, source Source, input Group) (Group, error) {
	input, err := normalizeGroup(input)
	if err != nil {
		return Group{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Group{}, err
	}
	defer tx.Rollback(ctx)
	if input.ExternalID != "" {
		var existing string
		err := tx.QueryRow(ctx, `SELECT id::text FROM directory_groups WHERE source_id=$1::uuid AND external_id=$2 AND deleted_at IS NULL`, source.ID, input.ExternalID).Scan(&existing)
		if err == nil {
			return Group{}, ErrConflict
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return Group{}, err
		}
	}
	var id string
	if err := tx.QueryRow(ctx, `
		INSERT INTO directory_groups(tenant_id,source_id,external_id,display_name)
		VALUES($1::uuid,$2::uuid,NULLIF($3,''),$4) RETURNING id::text`,
		source.TenantID, source.ID, input.ExternalID, input.DisplayName).Scan(&id); err != nil {
		return Group{}, mapPgError(err)
	}
	if err := replaceGroupMembers(ctx, tx, source, id, input.Members); err != nil {
		return Group{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Group{}, err
	}
	return r.GetGroup(ctx, source, id)
}

func (r *PostgresRepository) GetGroup(ctx context.Context, source Source, id string) (Group, error) {
	var group Group
	err := r.pool.QueryRow(ctx, `
		SELECT id::text,COALESCE(external_id,''),display_name,created_at,updated_at
		FROM directory_groups WHERE tenant_id=$1::uuid AND source_id=$2::uuid AND id::text=$3 AND deleted_at IS NULL`,
		source.TenantID, source.ID, strings.TrimSpace(id)).Scan(&group.ID, &group.ExternalID, &group.DisplayName, &group.CreatedAt, &group.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Group{}, ErrNotFound
	}
	if err != nil {
		return Group{}, err
	}
	members, err := r.loadGroupMembers(ctx, source, []string{group.ID})
	if err != nil {
		return Group{}, err
	}
	group.Members = members[group.ID]
	if group.Members == nil {
		group.Members = []GroupMember{}
	}
	return group, nil
}

func (r *PostgresRepository) ListGroups(ctx context.Context, source Source, filter GroupFilter, offset, limit int) ([]Group, int, error) {
	where := `tenant_id=$1::uuid AND source_id=$2::uuid AND deleted_at IS NULL`
	args := []any{source.TenantID, source.ID}
	if filter.DisplayName != "" {
		args = append(args, filter.DisplayName)
		where += fmt.Sprintf(" AND lower(display_name)=lower($%d)", len(args))
	}
	if filter.ExternalID != "" {
		args = append(args, filter.ExternalID)
		where += fmt.Sprintf(" AND external_id=$%d", len(args))
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM directory_groups WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if limit <= 0 || total == 0 {
		return []Group{}, total, nil
	}
	args = append(args, limit, max(offset, 0))
	rows, err := r.pool.Query(ctx, `SELECT id::text,COALESCE(external_id,''),display_name,created_at,updated_at FROM directory_groups WHERE `+where+
		fmt.Sprintf(` ORDER BY lower(display_name),id LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	groups := make([]Group, 0, min(limit, total))
	ids := make([]string, 0, min(limit, total))
	for rows.Next() {
		var group Group
		if err := rows.Scan(&group.ID, &group.ExternalID, &group.DisplayName, &group.CreatedAt, &group.UpdatedAt); err != nil {
			return nil, 0, err
		}
		groups = append(groups, group)
		ids = append(ids, group.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	members, err := r.loadGroupMembers(ctx, source, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range groups {
		groups[i].Members = members[groups[i].ID]
		if groups[i].Members == nil {
			groups[i].Members = []GroupMember{}
		}
	}
	return groups, total, nil
}

func (r *PostgresRepository) ReplaceGroup(ctx context.Context, source Source, id string, input Group) (Group, error) {
	input, err := normalizeGroup(input)
	if err != nil {
		return Group{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Group{}, err
	}
	defer tx.Rollback(ctx)
	var found string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM directory_groups WHERE tenant_id=$1::uuid AND source_id=$2::uuid AND id::text=$3 AND deleted_at IS NULL FOR UPDATE`, source.TenantID, source.ID, id).Scan(&found); errors.Is(err, pgx.ErrNoRows) {
		return Group{}, ErrNotFound
	} else if err != nil {
		return Group{}, err
	}
	if input.ExternalID != "" {
		var conflict bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM directory_groups WHERE source_id=$1::uuid AND external_id=$2 AND id::text<>$3 AND deleted_at IS NULL)`, source.ID, input.ExternalID, id).Scan(&conflict); err != nil {
			return Group{}, err
		}
		if conflict {
			return Group{}, ErrConflict
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE directory_groups SET external_id=NULLIF($1,''),display_name=$2,updated_at=clock_timestamp() WHERE tenant_id=$3::uuid AND id::text=$4`, input.ExternalID, input.DisplayName, source.TenantID, id); err != nil {
		return Group{}, mapPgError(err)
	}
	if err := replaceGroupMembers(ctx, tx, source, id, input.Members); err != nil {
		return Group{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Group{}, err
	}
	return r.GetGroup(ctx, source, id)
}

func (r *PostgresRepository) DeleteGroup(ctx context.Context, source Source, id string) error {
	command, err := r.pool.Exec(ctx, `
		UPDATE directory_groups SET deleted_at=clock_timestamp(),updated_at=clock_timestamp()
		WHERE tenant_id=$1::uuid AND source_id=$2::uuid AND id::text=$3 AND deleted_at IS NULL`, source.TenantID, source.ID, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) loadGroupMembers(ctx context.Context, source Source, groupIDs []string) (map[string][]GroupMember, error) {
	result := make(map[string][]GroupMember, len(groupIDs))
	if len(groupIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT dgm.group_id::text,su.id::text,p.display_name
		FROM directory_group_members dgm
		JOIN scim_users su ON su.id=dgm.scim_user_id AND su.tenant_id=dgm.tenant_id AND su.source_id=$2::uuid AND su.deleted_at IS NULL
		JOIN principals p ON p.id=su.principal_id AND p.tenant_id=su.tenant_id
		WHERE dgm.tenant_id=$1::uuid AND dgm.group_id::text=ANY($3::text[])
		ORDER BY dgm.group_id,su.id`, source.TenantID, source.ID, groupIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var groupID string
		var member GroupMember
		if err := rows.Scan(&groupID, &member.UserID, &member.DisplayName); err != nil {
			return nil, err
		}
		result[groupID] = append(result[groupID], member)
	}
	return result, rows.Err()
}

func replaceGroupMembers(ctx context.Context, tx pgx.Tx, source Source, groupID string, members []GroupMember) error {
	if len(members) > 10000 {
		return fmt.Errorf("%w: group supports at most 10000 direct members", ErrInvalid)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM directory_group_members WHERE tenant_id=$1::uuid AND group_id::text=$2`, source.TenantID, groupID); err != nil {
		return err
	}
	if len(members) == 0 {
		return nil
	}
	ids := make([]string, 0, len(members))
	seen := make(map[string]struct{}, len(members))
	for _, member := range members {
		id := strings.TrimSpace(member.UserID)
		if id == "" {
			return fmt.Errorf("%w: group member id is required", ErrInvalid)
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM scim_users WHERE tenant_id=$1::uuid AND source_id=$2::uuid AND id::text=ANY($3::text[]) AND deleted_at IS NULL`, source.TenantID, source.ID, ids).Scan(&count); err != nil {
		return err
	}
	if count != len(ids) {
		return fmt.Errorf("%w: every group member must reference an active User resource from the same SCIM source", ErrInvalid)
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO directory_group_members(tenant_id,group_id,scim_user_id)
		SELECT $1::uuid,$2::uuid,id FROM scim_users
		WHERE tenant_id=$1::uuid AND source_id=$3::uuid AND id::text=ANY($4::text[]) AND deleted_at IS NULL
		ON CONFLICT DO NOTHING`, source.TenantID, groupID, source.ID, ids)
	return err
}

func ensureUserUnique(ctx context.Context, tx pgx.Tx, source Source, id string, input User) error {
	var conflict bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM scim_users
			WHERE source_id=$1::uuid AND id::text<>$2 AND deleted_at IS NULL
			  AND (lower(user_name)=lower($3) OR ($4<>'' AND external_id=$4))
		)`, source.ID, id, input.UserName, input.ExternalID).Scan(&conflict); err != nil {
		return err
	}
	if conflict {
		return ErrConflict
	}
	return nil
}

func syncPrincipalIdentity(ctx context.Context, tx pgx.Tx, source Source, principalID string, user User) error {
	issuer := strings.TrimSpace(source.IdentityIssuer)
	if issuer == "" {
		return nil
	}
	subject := user.UserName
	if source.SubjectAttribute == "externalId" {
		subject = user.ExternalID
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return fmt.Errorf("%w: %s is required because this SCIM source links to OIDC", ErrInvalid, source.SubjectAttribute)
	}
	var existingPrincipal string
	err := tx.QueryRow(ctx, `SELECT principal_id::text FROM principal_identities WHERE tenant_id=$1::uuid AND issuer=$2 AND subject=$3`, source.TenantID, issuer, subject).Scan(&existingPrincipal)
	if err == nil && existingPrincipal != principalID {
		return ErrConflict
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if _, execErr := tx.Exec(ctx, `
		UPDATE principal_identities SET status='REVOKED',updated_at=clock_timestamp()
		WHERE tenant_id=$1::uuid AND principal_id=$2::uuid AND issuer=$3 AND subject<>$4 AND status='ACTIVE'`,
		source.TenantID, principalID, issuer, subject); execErr != nil {
		return execErr
	}
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = tx.Exec(ctx, `
			INSERT INTO principal_identities(tenant_id,principal_id,issuer,subject,status)
			VALUES($1::uuid,$2::uuid,$3,$4,'ACTIVE')`, source.TenantID, principalID, issuer, subject)
		return mapPgError(err)
	}
	_, err = tx.Exec(ctx, `UPDATE principal_identities SET status='ACTIVE',updated_at=clock_timestamp() WHERE tenant_id=$1::uuid AND issuer=$2 AND subject=$3`, source.TenantID, issuer, subject)
	return err
}

func normalizeUser(source Source, input User) (User, error) {
	input.UserName = strings.TrimSpace(input.UserName)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.ExternalID = strings.TrimSpace(input.ExternalID)
	if input.UserName == "" || len(input.UserName) > 320 {
		return User{}, fmt.Errorf("%w: userName is required and must be at most 320 characters", ErrInvalid)
	}
	if input.DisplayName == "" {
		input.DisplayName = input.UserName
	}
	if len(input.DisplayName) > 200 || len(input.ExternalID) > 2048 {
		return User{}, fmt.Errorf("%w: displayName or externalId exceeds supported bounds", ErrInvalid)
	}
	if source.IdentityIssuer != "" && source.SubjectAttribute == "externalId" && input.ExternalID == "" {
		return User{}, fmt.Errorf("%w: externalId is required because this SCIM source links it to OIDC subject", ErrInvalid)
	}
	return input, nil
}

func normalizeGroup(input Group) (Group, error) {
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.ExternalID = strings.TrimSpace(input.ExternalID)
	if input.DisplayName == "" || len(input.DisplayName) > 200 || len(input.ExternalID) > 2048 {
		return Group{}, fmt.Errorf("%w: displayName is required and group identifiers must be within supported bounds", ErrInvalid)
	}
	return input, nil
}

func principalStatus(active bool) string {
	if active {
		return "ACTIVE"
	}
	return "INACTIVE"
}

func mapPgError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrConflict
	}
	return err
}
