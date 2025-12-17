package store

import (
	"context"
	"fmt"
	"log"
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
        RETURNING id, email, name, picture;
    `
	row := s.db.QueryRow(ctx, query, email, name, googleID, picture)
	var user auth.UserProfile
	if err := row.Scan(&user.Id, &user.Email, &user.Name, &user.Picture); err != nil {
		return nil, fmt.Errorf("failed to create/update user: %w", err)
	}
	return &user, nil
}

func (s *PostgresStore) GetUserByGoogleID(ctx context.Context, googleID string) (*auth.UserProfile, error) {
	query := `SELECT id, email, name, picture FROM users WHERE google_id = $1`
	row := s.db.QueryRow(ctx, query, googleID)
	var user auth.UserProfile
	if err := row.Scan(&user.Id, &user.Email, &user.Name, &user.Picture); err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (s *PostgresStore) CreateMaterial(ctx context.Context, userID, matType, content, title string) (string, error) {
	log.Printf("[Store.CreateMaterial] Inserting material - UserID: %s, Type: %s, Title: %s", userID, matType, title)
	query := `
        INSERT INTO materials (user_id, type, content, title)
        VALUES ($1, $2, $3, $4)
        RETURNING id;
    `
	var materialID string
	err := s.db.QueryRow(ctx, query, userID, matType, content, title).Scan(&materialID)
	if err != nil {
		log.Printf("[Store.CreateMaterial] Insert failed: %v", err)
		return "", fmt.Errorf("failed to insert material: %w", err)
	}
	log.Printf("[Store.CreateMaterial] Material created with ID: %s", materialID)
	return materialID, nil
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

func (s *PostgresStore) GetDueMaterials(ctx context.Context, userID string, page, pageSize int32, searchQuery string, filterTags []string) ([]*learning.MaterialSummary, int32, error) {
	log.Printf("[Store.GetDueMaterials] Querying materials for userID: %s, page: %d, pageSize: %d, search: %s, tags: %v", userID, page, pageSize, searchQuery, filterTags)

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
		// Subquery to find material IDs that have ALL the specified tags (AND logic)
		// Or ANY tags (OR logic) - usually user expects OR or AND.
		// Let's implement OR logic for now as it's common filter behavior, or check user requirement.
		// User requirement "matchesSearch && matchesTags" in frontend implies AND logic implementation on frontend currently.
		// Detailed view of frontend filter: "selectedTags.every(tag => material.tags.includes(tag))" -> This is AND logic.
		// So we must implement AND logic.

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
	// Calculate offset
	offset := (page - 1) * pageSize

	// Add ordering and pagination limits
	paramCount++
	limitArgIdx := paramCount
	args = append(args, pageSize)

	paramCount++
	offsetArgIdx := paramCount
	args = append(args, offset)

	query := fmt.Sprintf(`
		SELECT m.id, m.title, COUNT(f.id) as due_count
		FROM materials m
		LEFT JOIN flashcards f ON m.id = f.material_id
		WHERE %s
		GROUP BY m.id, m.title
		ORDER BY m.created_at DESC
		LIMIT $%d OFFSET $%d;
	`, whereClause, limitArgIdx, offsetArgIdx)

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

func (s *PostgresStore) GetMaterialContent(ctx context.Context, userID, materialID string) (string, string, string, error) {
	log.Printf("[Store.GetMaterialContent] Fetching material: %s for user: %s", materialID, userID)
	query := `
		SELECT content, COALESCE(summary, ''), title
		FROM materials
		WHERE id = $1 AND user_id = $2;
	`
	var content, summary, title string
	err := s.db.QueryRow(ctx, query, materialID, userID).Scan(&content, &summary, &title)
	if err != nil {
		log.Printf("[Store.GetMaterialContent] Query failed: %v", err)
		return "", "", "", fmt.Errorf("failed to get material content: %w", err)
	}
	log.Printf("[Store.GetMaterialContent] Found material, content length: %d, has summary: %v", len(content), summary != "")
	return content, summary, title, nil
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
