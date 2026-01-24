package store

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/amityadav/landr/pkg/pb/auth"
	"github.com/amityadav/landr/pkg/pb/learning"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	db *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, connString string) (*PostgresStore, error) {
	db, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}
	if err := db.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Close() {
	s.db.Close()
}

func (s *PostgresStore) CreateUser(ctx context.Context, email, name, googleID, picture string) (*auth.UserProfile, error) {
	query := `
        INSERT INTO users (email, name, google_id, picture)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (google_id) DO UPDATE
        SET name = EXCLUDED.name, picture = EXCLUDED.picture, updated_at = NOW()
        RETURNING id, email, name, picture, COALESCE(is_admin, FALSE), COALESCE(is_blocked, FALSE);
    `
	row := s.db.QueryRow(ctx, query, email, name, googleID, picture)
	var user auth.UserProfile
	if err := row.Scan(&user.Id, &user.Email, &user.Name, &user.Picture, &user.IsAdmin, &user.IsBlocked); err != nil {
		return nil, fmt.Errorf("failed to create/update user: %w", err)
	}

	// Populate subscription status
	sub, err := s.GetSubscription(ctx, user.Id)
	if err == nil && sub.Plan == PlanPro && sub.Status == StatusActive {
		// Check if subscription is still valid (not expired)
		if sub.CurrentPeriodEnd == nil || sub.CurrentPeriodEnd.After(time.Now()) {
			user.IsPro = true
		}
	}

	return &user, nil
}

func (s *PostgresStore) GetUserByGoogleID(ctx context.Context, googleID string) (*auth.UserProfile, error) {
	query := `
		SELECT u.id, u.email, u.name, u.picture, COALESCE(u.is_admin, FALSE),
		       COALESCE(s.plan, 'FREE') = 'PRO' 
		       AND COALESCE(s.status, 'ACTIVE') IN ('ACTIVE', 'TRIALING')
		       AND (s.current_period_end IS NULL OR s.current_period_end > NOW()),
		       COALESCE(u.is_blocked, FALSE)
		FROM users u
		LEFT JOIN subscriptions s ON u.id = s.user_id
		WHERE u.google_id = $1
	`
	row := s.db.QueryRow(ctx, query, googleID)
	var user auth.UserProfile
	if err := row.Scan(&user.Id, &user.Email, &user.Name, &user.Picture, &user.IsAdmin, &user.IsPro, &user.IsBlocked); err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (s *PostgresStore) GetUserByID(ctx context.Context, userID string) (*auth.UserProfile, error) {
	query := `
		SELECT u.id, u.email, u.name, u.picture, COALESCE(u.is_admin, FALSE),
		       COALESCE(s.plan, 'PRO') = 'PRO' 
		       AND COALESCE(s.status, 'ACTIVE') IN ('ACTIVE', 'TRIALING')
		       AND (s.current_period_end IS NULL OR s.current_period_end > NOW()),
		       COALESCE(u.is_blocked, FALSE)
		FROM users u
		LEFT JOIN subscriptions s ON u.id = s.user_id
		WHERE u.id = $1
	`
	row := s.db.QueryRow(ctx, query, userID)
	var user auth.UserProfile
	if err := row.Scan(&user.Id, &user.Email, &user.Name, &user.Picture, &user.IsAdmin, &user.IsPro, &user.IsBlocked); err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// AdminUser represents a user for admin display with extra fields
type AdminUser struct {
	ID               string
	Email            string
	Name             string
	Picture          string
	IsAdmin          bool
	IsPro            bool
	IsBlocked        bool
	CreatedAt        time.Time
	MaterialCount    int
	CurrentPeriodEnd *time.Time
}

// GetAllUsersForAdmin returns paginated users with created_at for admin display
func (s *PostgresStore) GetAllUsersForAdmin(ctx context.Context, page, pageSize int, emailFilter string) ([]*AdminUser, int, error) {
	log.Printf("[Store.GetAllUsersForAdmin] Fetching users page=%d, pageSize=%d", page, pageSize)

	// Get total count
	var totalCount int
	// Build filter
	var args []interface{}
	whereClause := ""
	if emailFilter != "" {
		whereClause = "WHERE u.email ILIKE $1"
		args = append(args, "%"+emailFilter+"%")
	}

	countQuery := "SELECT COUNT(*) FROM users u " + whereClause
	if err := s.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	offset := (page - 1) * pageSize
	query := `
		SELECT u.id, u.email, u.name, u.picture, COALESCE(u.is_admin, FALSE),
		       COALESCE(s.plan, 'FREE') = 'PRO' 
		       AND COALESCE(s.status, 'ACTIVE') IN ('ACTIVE', 'TRIALING')
		       AND (s.current_period_end IS NULL OR s.current_period_end > NOW()) as is_pro,
		       COALESCE(u.is_blocked, FALSE),
		       u.created_at,
		       (SELECT COUNT(*) FROM materials m WHERE m.user_id = u.id AND (m.is_deleted = FALSE OR m.is_deleted IS NULL)) as material_count,
		       s.current_period_end
		FROM users u
		LEFT JOIN subscriptions s ON u.id = s.user_id
		` + whereClause + `
		ORDER BY u.created_at DESC
		LIMIT $` + strconv.Itoa(limitIdx) + ` OFFSET $` + strconv.Itoa(offsetIdx) + `
	`
	args = append(args, pageSize, offset)
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*AdminUser
	for rows.Next() {
		var user AdminUser
		if err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.Picture, &user.IsAdmin, &user.IsPro, &user.IsBlocked, &user.CreatedAt, &user.MaterialCount, &user.CurrentPeriodEnd); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}
	log.Printf("[Store.GetAllUsersForAdmin] Found %d users (total: %d)", len(users), totalCount)
	return users, totalCount, nil
}

// GetAllUsers returns all users in the system
func (s *PostgresStore) GetAllUsers(ctx context.Context) ([]*auth.UserProfile, error) {
	log.Printf("[Store.GetAllUsers] Fetching all users")
	query := `SELECT id, email, name, picture, COALESCE(is_admin, FALSE) FROM users ORDER BY created_at DESC`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*auth.UserProfile
	for rows.Next() {
		var user auth.UserProfile
		if err := rows.Scan(&user.Id, &user.Email, &user.Name, &user.Picture, &user.IsAdmin); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}
	log.Printf("[Store.GetAllUsers] Found %d users", len(users))
	return users, nil
}

// SetUserAdminStatus sets the admin status for a user by email
func (s *PostgresStore) SetUserAdminStatus(ctx context.Context, email string, isAdmin bool) error {
	log.Printf("[Store.SetUserAdminStatus] Setting admin=%v for email: %s", isAdmin, email)
	query := `UPDATE users SET is_admin = $1, updated_at = NOW() WHERE email = $2`
	result, err := s.db.Exec(ctx, query, isAdmin, email)
	if err != nil {
		return fmt.Errorf("failed to update admin status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found with email: %s", email)
	}
	log.Printf("[Store.SetUserAdminStatus] Admin status updated successfully")
	return nil
}

// SetUserBlockStatus sets the blocked status for a user by email
func (s *PostgresStore) SetUserBlockStatus(ctx context.Context, email string, isBlocked bool) error {
	log.Printf("[Store.SetUserBlockStatus] Setting blocked=%v for email: %s", isBlocked, email)
	query := `UPDATE users SET is_blocked = $1, updated_at = NOW() WHERE email = $2`
	result, err := s.db.Exec(ctx, query, isBlocked, email)
	if err != nil {
		return fmt.Errorf("failed to update blocked status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found with email: %s", email)
	}
	log.Printf("[Store.SetUserBlockStatus] Blocked status updated successfully")
	return nil
}

// SetUserProStatus sets the pro status for a user by ID
func (s *PostgresStore) SetUserProStatus(ctx context.Context, userID string, isPro bool, days int) error {
	log.Printf("[Store.SetUserProStatus] Setting pro=%v for userID: %s, days=%d", isPro, userID, days)

	var plan SubscriptionPlan
	var status SubscriptionStatus

	if isPro {
		plan = PlanPro
		status = StatusActive
	} else {
		plan = PlanFree
		status = StatusActive
	}

	var periodEnd time.Time
	if isPro {
		// Set days for admin-granted pro
		periodEnd = time.Now().AddDate(0, 0, days)
	} else {
		// Set to now (expired) if removing pro
		periodEnd = time.Now()
	}

	// Use UpsertSubscription
	sub := &Subscription{
		UserID:                 userID,
		Plan:                   plan,
		Status:                 status,
		CurrentPeriodEnd:       &periodEnd,
		RazorpaySubscriptionID: "manual_admin_set",
	}

	if err := s.UpsertSubscription(ctx, sub); err != nil {
		return err
	}

	// Disable feed if downgrading to free
	if !isPro {
		log.Printf("[Store.SetUserProStatus] Disabling feed for user %s due to downgrade", userID)
		query := `UPDATE users SET feed_enabled = FALSE, updated_at = NOW() WHERE id = $1`
		if _, err := s.db.Exec(ctx, query, userID); err != nil {
			return fmt.Errorf("failed to disable feed during downgrade: %w", err)
		}
	}

	return nil
}

func (s *PostgresStore) CreateMaterial(ctx context.Context, userID, matType, content, title, sourceURL string) (string, error) {
	log.Printf("[Store.CreateMaterial] Inserting material - UserID: %s, Type: %s, Title: %s", userID, matType, title)
	query := `
        INSERT INTO materials (user_id, type, content, title, source_url)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id;
    `
	var materialID string
	err := s.db.QueryRow(ctx, query, userID, matType, content, title, sourceURL).Scan(&materialID)
	if err != nil {
		log.Printf("[Store.CreateMaterial] Insert failed: %v", err)
		return "", fmt.Errorf("failed to insert material: %w", err)
	}
	log.Printf("[Store.CreateMaterial] Material created with ID: %s", materialID)
	return materialID, nil
}

func (s *PostgresStore) GetMaterialBySourceURL(ctx context.Context, userID, sourceURL string) (string, error) {
	query := `SELECT id FROM materials WHERE user_id = $1 AND source_url = $2 AND is_deleted = FALSE LIMIT 1;`
	var id string
	err := s.db.QueryRow(ctx, query, userID, sourceURL).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *PostgresStore) SoftDeleteMaterial(ctx context.Context, userID, materialID string) error {
	log.Printf("[Store.SoftDeleteMaterial] Soft deleting material: %s for user: %s", materialID, userID)
	query := `
		UPDATE materials 
		SET is_deleted = TRUE, deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND is_deleted = FALSE;
	`
	result, err := s.db.Exec(ctx, query, materialID, userID)
	if err != nil {
		log.Printf("[Store.SoftDeleteMaterial] Delete failed: %v", err)
		return fmt.Errorf("failed to delete material: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("material not found or already deleted")
	}
	log.Printf("[Store.SoftDeleteMaterial] Material soft deleted successfully")
	return nil
}

func (s *PostgresStore) CreateTag(ctx context.Context, userID, name string) (string, error) {
	query := `
		INSERT INTO tags (user_id, name) VALUES ($1, $2)
		ON CONFLICT (user_id, name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id;
	`
	var id string
	err := s.db.QueryRow(ctx, query, userID, name).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to create tag: %w", err)
	}
	return id, nil
}

func (s *PostgresStore) GetTags(ctx context.Context, userID string) ([]string, error) {
	query := `SELECT name FROM tags WHERE user_id = $1 ORDER BY name`
	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tags = append(tags, name)
	}
	return tags, nil
}

func (s *PostgresStore) AddMaterialTags(ctx context.Context, materialID string, tagIDs []string) error {
	for _, tagID := range tagIDs {
		query := `INSERT INTO material_tags (material_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
		if _, err := s.db.Exec(ctx, query, materialID, tagID); err != nil {
			return fmt.Errorf("failed to link tag: %w", err)
		}
	}
	return nil
}

func (s *PostgresStore) GetMaterialTags(ctx context.Context, materialID string) ([]string, error) {
	query := `
		SELECT t.name 
		FROM tags t
		JOIN material_tags mt ON t.id = mt.tag_id
		WHERE mt.material_id = $1
	`
	rows, err := s.db.Query(ctx, query, materialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tags = append(tags, name)
	}
	return tags, nil
}

func (s *PostgresStore) CreateFlashcards(ctx context.Context, materialID string, cards []*learning.Flashcard) error {
	log.Printf("[Store.CreateFlashcards] Inserting %d flashcards for material: %s", len(cards), materialID)
	for i, card := range cards {
		query := `
            INSERT INTO flashcards (material_id, question, answer, stage, next_review_at)
            VALUES ($1, $2, $3, $4, NOW());
        `
		_, err := s.db.Exec(ctx, query, materialID, card.Question, card.Answer, 0)
		if err != nil {
			log.Printf("[Store.CreateFlashcards] Failed to insert flashcard %d: %v", i, err)
			return fmt.Errorf("failed to insert flashcard: %w", err)
		}
	}
	log.Printf("[Store.CreateFlashcards] All flashcards inserted successfully")
	return nil
}

func (s *PostgresStore) GetFlashcard(ctx context.Context, id string) (*learning.Flashcard, error) {
	log.Printf("[Store.GetFlashcard] Querying flashcard: %s", id)
	query := `
		SELECT f.id, f.question, f.answer, f.stage, f.next_review_at, m.title, m.id
		FROM flashcards f
		JOIN materials m ON f.material_id = m.id
		WHERE f.id = $1;
	`
	row := s.db.QueryRow(ctx, query, id)

	var card learning.Flashcard
	var title string
	var matID string
	var nextReviewAt time.Time

	if err := row.Scan(&card.Id, &card.Question, &card.Answer, &card.Stage, &nextReviewAt, &title, &matID); err != nil {
		log.Printf("[Store.GetFlashcard] Query failed: %v", err)
		return nil, fmt.Errorf("failed to query flashcard: %w", err)
	}

	card.MaterialTitle = title

	tags, err := s.GetMaterialTags(ctx, matID)
	if err != nil {
		log.Printf("[Store.GetFlashcard] Failed to get tags: %v", err)
		tags = []string{}
	}
	card.Tags = tags

	log.Printf("[Store.GetFlashcard] Found flashcard at stage %d", card.Stage)
	return &card, nil
}

func (s *PostgresStore) GetDueFlashcards(ctx context.Context, userID, materialID string) ([]*learning.Flashcard, error) {
	log.Printf("[Store.GetDueFlashcards] Querying flashcards for userID: %s, materialID: %s", userID, materialID)
	query := `
        SELECT f.id, f.question, f.answer, f.stage, m.title, m.id
        FROM flashcards f
        JOIN materials m ON f.material_id = m.id
        WHERE m.user_id = $1 AND m.id = $2 AND (m.is_deleted = FALSE OR m.is_deleted IS NULL)
        ORDER BY f.id ASC;
    `
	rows, err := s.db.Query(ctx, query, userID, materialID)
	if err != nil {
		log.Printf("[Store.GetDueFlashcards] Query failed: %v", err)
		return nil, fmt.Errorf("failed to query flashcards: %w", err)
	}
	defer rows.Close()

	var flashcards []*learning.Flashcard
	for rows.Next() {
		var card learning.Flashcard
		var title string
		var matID string
		if err := rows.Scan(&card.Id, &card.Question, &card.Answer, &card.Stage, &title, &matID); err != nil {
			log.Printf("[Store.GetDueFlashcards] Scan failed: %v", err)
			return nil, fmt.Errorf("failed to scan flashcard: %w", err)
		}
		card.MaterialTitle = title

		tags, err := s.GetMaterialTags(ctx, matID)
		if err != nil {
			log.Printf("[Store.GetDueFlashcards] Failed to get tags: %v", err)
			tags = []string{}
		}
		card.Tags = tags

		flashcards = append(flashcards, &card)
	}

	return flashcards, nil
}

func (s *PostgresStore) UpdateFlashcardContent(ctx context.Context, id, question, answer string) error {
	log.Printf("[Store.UpdateFlashcardContent] Updating flashcard: %s", id)
	query := `
		UPDATE flashcards
		SET question = $1, answer = $2, updated_at = NOW()
		WHERE id = $3;
	`
	result, err := s.db.Exec(ctx, query, question, answer, id)
	if err != nil {
		log.Printf("[Store.UpdateFlashcardContent] Update failed: %v", err)
		return fmt.Errorf("failed to update flashcard content: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("flashcard not found")
	}
	log.Printf("[Store.UpdateFlashcardContent] Flashcard content updated successfully")
	return nil
}

func (s *PostgresStore) GetDueMaterials(ctx context.Context, userID string, page, pageSize int32, searchQuery string, filterTags []string, onlyDue bool) ([]*learning.MaterialSummary, int32, error) {
	log.Printf("[Store.GetDueMaterials] Querying materials for userID: %s, page: %d, pageSize: %d, search: %s, tags: %v, onlyDue: %v", userID, page, pageSize, searchQuery, filterTags, onlyDue)

	// Base conditions
	whereClause := "m.user_id = $1 AND (m.is_deleted = FALSE OR m.is_deleted IS NULL)"
	args := []interface{}{userID}
	paramCount := 1

	// Add search query filter
	if searchQuery != "" {
		paramCount++
		whereClause += fmt.Sprintf(" AND m.title ILIKE $%d", paramCount)
		args = append(args, "%"+searchQuery+"%")
	}

	// Add tag filter
	if len(filterTags) > 0 {
		paramCount++
		whereClause += fmt.Sprintf(` AND m.id IN (
			SELECT mt.material_id 
			FROM material_tags mt 
			JOIN tags t ON mt.tag_id = t.id 
			WHERE t.name = ANY($%d)
			GROUP BY mt.material_id 
			HAVING COUNT(DISTINCT t.name) = $%d
		)`, paramCount, paramCount+1)
		args = append(args, filterTags, len(filterTags))
		paramCount++
	}

	// Subquery for due count
	dueCountSubquery := `(SELECT COUNT(f.id) FROM flashcards f WHERE f.material_id = m.id AND f.next_review_at <= NOW())`

	// Add only_due filter
	if onlyDue {
		whereClause += fmt.Sprintf(" AND %s > 0", dueCountSubquery)
	}

	// 1. Get total count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT m.id)
		FROM materials m
		WHERE %s
	`, whereClause)

	var totalCount int32
	if err := s.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count materials: %w", err)
	}

	// 2. Get paginated results
	offset := (page - 1) * pageSize

	paramCount++
	limitArgIdx := paramCount
	args = append(args, pageSize)

	paramCount++
	offsetArgIdx := paramCount
	args = append(args, offset)

	query := fmt.Sprintf(`
		SELECT m.id, m.title, %s as due_count
		FROM materials m
		WHERE %s
		ORDER BY m.created_at DESC
		LIMIT $%d OFFSET $%d;
	`, dueCountSubquery, whereClause, limitArgIdx, offsetArgIdx)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query materials: %w", err)
	}
	defer rows.Close()

	var materials []*learning.MaterialSummary
	for rows.Next() {
		var m learning.MaterialSummary
		if err := rows.Scan(&m.Id, &m.Title, &m.DueCount); err != nil {
			return nil, 0, fmt.Errorf("failed to scan material: %w", err)
		}

		tags, err := s.GetMaterialTags(ctx, m.Id)
		if err != nil {
			log.Printf("[Store.GetDueMaterials] Failed to get tags: %v", err)
			tags = []string{}
		}
		m.Tags = tags

		materials = append(materials, &m)
	}

	log.Printf("[Store.GetDueMaterials] Found %d materials (total: %d)", len(materials), totalCount)
	return materials, totalCount, nil
}

func (s *PostgresStore) GetDueFlashcardsCount(ctx context.Context, userID string) (int32, error) {
	log.Printf("[Store.GetDueFlashcardsCount] Counting due flashcards for userID: %s", userID)
	query := `
		SELECT COUNT(f.id)
		FROM flashcards f
		JOIN materials m ON f.material_id = m.id
		WHERE m.user_id = $1 AND f.next_review_at <= NOW() AND (m.is_deleted = FALSE OR m.is_deleted IS NULL);
	`
	var count int32
	if err := s.db.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		log.Printf("[Store.GetDueFlashcardsCount] Query failed: %v", err)
		return 0, fmt.Errorf("failed to count due flashcards: %w", err)
	}
	log.Printf("[Store.GetDueFlashcardsCount] Found %d due flashcards", count)
	return count, nil
}
func (s *PostgresStore) GetNotificationData(ctx context.Context, userID string) (int32, int32, string, error) {
	log.Printf("[Store.GetNotificationData] Fetching data for userID: %s", userID)

	// 1. Get due flashcards count
	flashcardQuery := `
		SELECT COUNT(f.id)
		FROM flashcards f
		JOIN materials m ON f.material_id = m.id
		WHERE m.user_id = $1 AND f.next_review_at <= NOW() AND (m.is_deleted = FALSE OR m.is_deleted IS NULL);
	`
	var flashcardsCount int32
	if err := s.db.QueryRow(ctx, flashcardQuery, userID).Scan(&flashcardsCount); err != nil {
		return 0, 0, "", fmt.Errorf("failed to count due flashcards: %w", err)
	}

	// 2. Get due materials count and first title
	// A material is due if it has at least one due flashcard
	materialQuery := `
		SELECT COUNT(DISTINCT m.id), COALESCE(MAX(m.title) FILTER (WHERE m.id = (
			SELECT sub_m.id 
			FROM materials sub_m
			JOIN flashcards f ON sub_m.id = f.material_id
			WHERE sub_m.user_id = $1 AND f.next_review_at <= NOW() AND (sub_m.is_deleted = FALSE OR sub_m.is_deleted IS NULL)
			ORDER BY f.next_review_at ASC
			LIMIT 1
		)), '')
		FROM materials m
		JOIN flashcards f ON m.id = f.material_id
		WHERE m.user_id = $1 AND f.next_review_at <= NOW() AND (m.is_deleted = FALSE OR m.is_deleted IS NULL);
	`
	var materialsCount int32
	var firstTitle string
	if err := s.db.QueryRow(ctx, materialQuery, userID).Scan(&materialsCount, &firstTitle); err != nil {
		return 0, 0, "", fmt.Errorf("failed to get material notification data: %w", err)
	}

	log.Printf("[Store.GetNotificationData] flashcards: %d, materials: %d, firstTitle: %s", flashcardsCount, materialsCount, firstTitle)
	return flashcardsCount, materialsCount, firstTitle, nil
}

func (s *PostgresStore) UpdateFlashcard(ctx context.Context, id string, stage int32, nextReviewAt time.Time) error {
	log.Printf("[Store.UpdateFlashcard] Updating flashcard: %s, Stage: %d, NextReviewAt: %v", id, stage, nextReviewAt)
	query := `
        UPDATE flashcards
        SET stage = $1, next_review_at = $2, updated_at = NOW()
        WHERE id = $3;
    `
	_, err := s.db.Exec(ctx, query, stage, nextReviewAt, id)
	if err != nil {
		log.Printf("[Store.UpdateFlashcard] Update failed: %v", err)
		return fmt.Errorf("failed to update flashcard: %w", err)
	}
	log.Printf("[Store.UpdateFlashcard] Flashcard updated successfully")
	return nil
}

func (s *PostgresStore) GetMaterialContent(ctx context.Context, userID, materialID string) (string, string, string, string, string, error) {
	log.Printf("[Store.GetMaterialContent] Fetching material: %s for user: %s", materialID, userID)
	query := `
		SELECT content, COALESCE(summary, ''), title, type, COALESCE(source_url, '')
		FROM materials
		WHERE id = $1 AND user_id = $2;
	`
	var content, summary, title, materialType, sourceURL string
	err := s.db.QueryRow(ctx, query, materialID, userID).Scan(&content, &summary, &title, &materialType, &sourceURL)
	if err != nil {
		log.Printf("[Store.GetMaterialContent] Query failed: %v", err)
		return "", "", "", "", "", fmt.Errorf("failed to get material content: %w", err)
	}
	log.Printf("[Store.GetMaterialContent] Found material, content length: %d, has summary: %v, type: %s", len(content), summary != "", materialType)
	return content, summary, title, materialType, sourceURL, nil
}

func (s *PostgresStore) UpdateMaterialSummary(ctx context.Context, materialID, summary string) error {
	log.Printf("[Store.UpdateMaterialSummary] Updating summary for material: %s", materialID)
	query := `
		UPDATE materials
		SET summary = $1, updated_at = NOW()
		WHERE id = $2;
	`
	_, err := s.db.Exec(ctx, query, summary, materialID)
	if err != nil {
		log.Printf("[Store.UpdateMaterialSummary] Update failed: %v", err)
		return fmt.Errorf("failed to update material summary: %w", err)
	}
	log.Printf("[Store.UpdateMaterialSummary] Summary updated successfully")
	return nil
}

// =======================
// Feed Methods
// =======================

// FeedPreferences represents user's feed configuration
type FeedPreferences struct {
	InterestPrompt string
	FeedEnabled    bool
	FeedEvalPrompt string
}

// GetFeedPreferences fetches the user's feed preferences
func (s *PostgresStore) GetFeedPreferences(ctx context.Context, userID string) (*FeedPreferences, error) {
	log.Printf("[Store.GetFeedPreferences] Fetching for userID: %s", userID)
	query := `SELECT COALESCE(interest_prompt, ''), COALESCE(feed_enabled, FALSE), COALESCE(feed_eval_prompt, '') FROM users WHERE id = $1`
	var prefs FeedPreferences
	err := s.db.QueryRow(ctx, query, userID).Scan(&prefs.InterestPrompt, &prefs.FeedEnabled, &prefs.FeedEvalPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed preferences: %w", err)
	}
	return &prefs, nil
}

// UpdateFeedPreferences updates the user's feed preferences
func (s *PostgresStore) UpdateFeedPreferences(ctx context.Context, userID, interestPrompt, evalPrompt string, feedEnabled bool) error {
	log.Printf("[Store.UpdateFeedPreferences] Updating for userID: %s, enabled: %v", userID, feedEnabled)
	query := `UPDATE users SET interest_prompt = $1, feed_eval_prompt = $2, feed_enabled = $3, updated_at = NOW() WHERE id = $4`
	_, err := s.db.Exec(ctx, query, interestPrompt, evalPrompt, feedEnabled, userID)
	if err != nil {
		return fmt.Errorf("failed to update feed preferences: %w", err)
	}
	return nil
}

// DailyArticle represents an article recommended to a user
type DailyArticle struct {
	ID             string
	Title          string
	URL            string
	Snippet        string
	RelevanceScore float64
	SuggestedDate  time.Time
	CreatedAt      time.Time
	Provider       string
}

// StoreDailyArticle stores a single daily article for a user
func (s *PostgresStore) StoreDailyArticle(ctx context.Context, userID string, article *DailyArticle) error {
	log.Printf("[Store.StoreDailyArticle] Storing article: %s for user: %s (provider: %s)", article.Title, userID, article.Provider)
	query := `
		INSERT INTO daily_articles (user_id, title, url, snippet, suggested_date, relevance_score, provider)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := s.db.Exec(ctx, query, userID, article.Title, article.URL, article.Snippet, article.SuggestedDate, article.RelevanceScore, article.Provider)
	if err != nil {
		return fmt.Errorf("failed to store daily article: %w", err)
	}
	return nil
}

// GetDailyArticles fetches articles for a specific date
func (s *PostgresStore) GetDailyArticles(ctx context.Context, userID string, date time.Time) ([]*DailyArticle, error) {
	log.Printf("[Store.GetDailyArticles] Fetching for userID: %s, date param: %v (Unix: %d)", userID, date, date.Unix())
	query := `
		SELECT id, title, url, snippet, relevance_score, suggested_date, created_at, provider
		FROM daily_articles
		WHERE user_id = $1 AND suggested_date = $2
		ORDER BY relevance_score DESC
	`
	rows, err := s.db.Query(ctx, query, userID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily articles: %w", err)
	}
	defer rows.Close()

	var articles []*DailyArticle
	for rows.Next() {
		var a DailyArticle
		if err := rows.Scan(&a.ID, &a.Title, &a.URL, &a.Snippet, &a.RelevanceScore, &a.SuggestedDate, &a.CreatedAt, &a.Provider); err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		articles = append(articles, &a)
	}
	log.Printf("[Store.GetDailyArticles] Found %d articles", len(articles))
	return articles, nil
}

// CalendarDay represents a day with article count for the calendar view
type CalendarDay struct {
	Date         time.Time
	ArticleCount int32
}

// GetFeedCalendarStatus fetches dates that have articles for a given month
func (s *PostgresStore) GetFeedCalendarStatus(ctx context.Context, userID string, year, month int) ([]*CalendarDay, error) {
	log.Printf("[Store.GetFeedCalendarStatus] Fetching for userID: %s, %d-%02d", userID, year, month)
	query := `
		SELECT suggested_date, COUNT(id) as article_count
		FROM daily_articles
		WHERE user_id = $1 AND EXTRACT(YEAR FROM suggested_date) = $2 AND EXTRACT(MONTH FROM suggested_date) = $3
		GROUP BY suggested_date
		ORDER BY suggested_date
	`
	rows, err := s.db.Query(ctx, query, userID, year, month)
	if err != nil {
		return nil, fmt.Errorf("failed to query calendar status: %w", err)
	}
	defer rows.Close()

	var days []*CalendarDay
	for rows.Next() {
		var d CalendarDay
		if err := rows.Scan(&d.Date, &d.ArticleCount); err != nil {
			return nil, fmt.Errorf("failed to scan calendar day: %w", err)
		}
		days = append(days, &d)
	}
	log.Printf("[Store.GetFeedCalendarStatus] Found %d days with articles", len(days))
	return days, nil
}

// GetUsersWithFeedEnabled fetches all user IDs with feed enabled
func (s *PostgresStore) GetUsersWithFeedEnabled(ctx context.Context) ([]string, error) {
	log.Printf("[Store.GetUsersWithFeedEnabled] Fetching users...")
	query := `SELECT id FROM users WHERE feed_enabled = TRUE AND interest_prompt IS NOT NULL AND interest_prompt != ''`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users with feed enabled: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, id)
	}
	log.Printf("[Store.GetUsersWithFeedEnabled] Found %d users", len(userIDs))
	return userIDs, nil
}

// GetUserByEmail fetches a user ID by email address
func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (string, error) {
	log.Printf("[Store.GetUserByEmail] Looking up user by email: %s", email)
	query := `SELECT id FROM users WHERE email = $1`
	var userID string
	err := s.db.QueryRow(ctx, query, email).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("user not found: %w", err)
	}
	return userID, nil
}

// ArticleURLExists checks if an article URL already exists for this user
func (s *PostgresStore) ArticleURLExists(ctx context.Context, userID, url string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM daily_articles WHERE user_id = $1 AND url = $2)`
	var exists bool
	err := s.db.QueryRow(ctx, query, userID, url).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check article existence: %w", err)
	}
	return exists, nil
}

// ============================================
// Device Tokens (FCM Push Notifications)
// ============================================

// SaveDeviceToken stores or updates a device token for push notifications
func (s *PostgresStore) SaveDeviceToken(ctx context.Context, userID, token, platform string) error {
	query := `
		INSERT INTO device_tokens (user_id, token, platform, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_id, token) DO UPDATE SET updated_at = NOW()
	`
	_, err := s.db.Exec(ctx, query, userID, token, platform)
	if err != nil {
		return fmt.Errorf("failed to save device token: %w", err)
	}
	log.Printf("[Store.SaveDeviceToken] Saved token for user %s (platform: %s)", userID, platform)
	return nil
}

// GetDeviceTokens returns all device tokens for a user
func (s *PostgresStore) GetDeviceTokens(ctx context.Context, userID string) ([]string, error) {
	query := `SELECT token FROM device_tokens WHERE user_id = $1`
	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device tokens: %w", err)
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

// GetAllUsersWithTokens returns user IDs that have registered device tokens
func (s *PostgresStore) GetAllUsersWithTokens(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT user_id FROM device_tokens`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get users with tokens: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			continue
		}
		userIDs = append(userIDs, userID)
	}
	return userIDs, nil
}

// DeleteDeviceToken removes a device token (for logout)
func (s *PostgresStore) DeleteDeviceToken(ctx context.Context, userID, token string) error {
	query := `DELETE FROM device_tokens WHERE user_id = $1 AND token = $2`
	_, err := s.db.Exec(ctx, query, userID, token)
	return err
}

// =======================
// Settings Methods
// =======================

// GetSetting retrieves a setting value by key
func (s *PostgresStore) GetSetting(ctx context.Context, key string) ([]byte, error) {
	query := `SELECT value FROM settings WHERE key = $1`
	var value []byte
	err := s.db.QueryRow(ctx, query, key).Scan(&value)
	if err != nil {
		return nil, fmt.Errorf("failed to get setting '%s': %w", key, err)
	}
	return value, nil
}

// SetSetting creates or updates a setting
func (s *PostgresStore) SetSetting(ctx context.Context, key string, value []byte, description string) error {
	query := `
		INSERT INTO settings (key, value, description)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE SET value = $2, description = $3, updated_at = NOW()
	`
	_, err := s.db.Exec(ctx, query, key, value, description)
	if err != nil {
		return fmt.Errorf("failed to set setting '%s': %w", key, err)
	}
	return nil
}

// GetAllSettings retrieves all settings from the database
func (s *PostgresStore) GetAllSettings(ctx context.Context) ([]SettingRow, error) {
	query := `SELECT key, value, COALESCE(description, '') FROM settings ORDER BY key`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all settings: %w", err)
	}
	defer rows.Close()

	var settings []SettingRow
	for rows.Next() {
		var row SettingRow
		if err := rows.Scan(&row.Key, &row.Value, &row.Description); err != nil {
			return nil, fmt.Errorf("failed to scan setting row: %w", err)
		}
		settings = append(settings, row)
	}
	return settings, nil
}
