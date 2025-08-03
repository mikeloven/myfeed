package services

import (
	"database/sql"
	"fmt"
	"myfeed/database"
	"myfeed/models"
)

type FolderService struct {
	db *database.DB
}

func NewFolderService(db *database.DB) *FolderService {
	return &FolderService{db: db}
}

func (fs *FolderService) CreateFolder(name string, parentID *int) (*models.Folder, error) {
	if name == "" {
		return nil, fmt.Errorf("folder name cannot be empty")
	}

	// Check if folder with same name exists at the same level
	var count int
	checkQuery := `SELECT COUNT(*) FROM folders WHERE name = ? AND parent_id IS ?`
	err := fs.db.QueryRow(checkQuery, name, parentID).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to check folder existence: %v", err)
	}
	
	if count > 0 {
		return nil, fmt.Errorf("folder with name '%s' already exists", name)
	}

	// Get the next position for this folder
	var maxPosition sql.NullInt64
	posQuery := `SELECT MAX(position) FROM folders WHERE parent_id IS ?`
	err = fs.db.QueryRow(posQuery, parentID).Scan(&maxPosition)
	if err != nil {
		return nil, fmt.Errorf("failed to get folder position: %v", err)
	}

	position := 0
	if maxPosition.Valid {
		position = int(maxPosition.Int64) + 1
	}

	// Insert the folder
	query := `
		INSERT INTO folders (name, parent_id, position)
		VALUES (?, ?, ?)
	`
	
	result, err := fs.db.Exec(query, name, parentID, position)
	if err != nil {
		return nil, fmt.Errorf("failed to create folder: %v", err)
	}

	folderID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get folder ID: %v", err)
	}

	return fs.GetFolderByID(int(folderID))
}

func (fs *FolderService) GetFolderByID(id int) (*models.Folder, error) {
	query := `
		SELECT id, name, parent_id, position, created_at
		FROM folders WHERE id = ?
	`
	
	folder := &models.Folder{}
	err := fs.db.QueryRow(query, id).Scan(
		&folder.ID, &folder.Name, &folder.ParentID, &folder.Position, &folder.CreatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return folder, nil
}

func (fs *FolderService) GetAllFolders() ([]models.Folder, error) {
	query := `
		SELECT id, name, parent_id, position, created_at
		FROM folders ORDER BY parent_id, position, name
	`
	
	rows, err := fs.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []models.Folder
	for rows.Next() {
		folder := models.Folder{}
		err := rows.Scan(
			&folder.ID, &folder.Name, &folder.ParentID, &folder.Position, &folder.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		folders = append(folders, folder)
	}
	
	return folders, nil
}

func (fs *FolderService) UpdateFolder(id int, name string) (*models.Folder, error) {
	if name == "" {
		return nil, fmt.Errorf("folder name cannot be empty")
	}

	// Check if folder exists
	existingFolder, err := fs.GetFolderByID(id)
	if err != nil {
		return nil, fmt.Errorf("folder not found: %v", err)
	}

	// Check if another folder with same name exists at the same level
	var count int
	checkQuery := `SELECT COUNT(*) FROM folders WHERE name = ? AND parent_id IS ? AND id != ?`
	err = fs.db.QueryRow(checkQuery, name, existingFolder.ParentID, id).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to check folder existence: %v", err)
	}
	
	if count > 0 {
		return nil, fmt.Errorf("folder with name '%s' already exists", name)
	}

	// Update the folder
	query := `UPDATE folders SET name = ? WHERE id = ?`
	_, err = fs.db.Exec(query, name, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update folder: %v", err)
	}

	return fs.GetFolderByID(id)
}

func (fs *FolderService) DeleteFolder(id int) error {
	// Check if folder has any feeds
	var feedCount int
	feedQuery := `SELECT COUNT(*) FROM feeds WHERE folder_id = ?`
	err := fs.db.QueryRow(feedQuery, id).Scan(&feedCount)
	if err != nil {
		return fmt.Errorf("failed to check folder feeds: %v", err)
	}

	if feedCount > 0 {
		return fmt.Errorf("cannot delete folder: it contains %d feeds", feedCount)
	}

	// Check if folder has any subfolders
	var subfolderCount int
	subQuery := `SELECT COUNT(*) FROM folders WHERE parent_id = ?`
	err = fs.db.QueryRow(subQuery, id).Scan(&subfolderCount)
	if err != nil {
		return fmt.Errorf("failed to check subfolders: %v", err)
	}

	if subfolderCount > 0 {
		return fmt.Errorf("cannot delete folder: it contains %d subfolders", subfolderCount)
	}

	// Delete the folder
	query := `DELETE FROM folders WHERE id = ?`
	result, err := fs.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete folder: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (fs *FolderService) MoveFeedsToFolder(feedIDs []int, folderID *int) error {
	// Validate folder exists if folderID is provided
	if folderID != nil {
		_, err := fs.GetFolderByID(*folderID)
		if err != nil {
			return fmt.Errorf("target folder not found: %v", err)
		}
	}

	// Update feeds
	query := `UPDATE feeds SET folder_id = ? WHERE id = ?`
	for _, feedID := range feedIDs {
		_, err := fs.db.Exec(query, folderID, feedID)
		if err != nil {
			return fmt.Errorf("failed to move feed %d: %v", feedID, err)
		}
	}

	return nil
}

func (fs *FolderService) GetFeedsInFolder(folderID *int) ([]models.Feed, error) {
	query := `
		SELECT id, url, title, description, folder_id, created_at, updated_at, 
		       last_fetch, health, error_count
		FROM feeds WHERE folder_id IS ? ORDER BY title
	`
	
	rows, err := fs.db.Query(query, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []models.Feed
	for rows.Next() {
		feed := models.Feed{}
		err := rows.Scan(
			&feed.ID, &feed.URL, &feed.Title, &feed.Description, &feed.FolderID,
			&feed.CreatedAt, &feed.UpdatedAt, &feed.LastFetch, &feed.Health, &feed.ErrorCount,
		)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}
	
	return feeds, nil
}