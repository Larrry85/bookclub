package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var (
	DB       *sql.DB
	Sessions = make(map[string]string) // Session ID -> User ID
)

func Init() {
	var err error
	DB, err = sql.Open("sqlite3", "./user.db")
	if err != nil {
		log.Fatal(err)
	}

	// Execute schema creation
	schema := `
	CREATE TABLE IF NOT EXISTS User (
		UserID INTEGER PRIMARY KEY AUTOINCREMENT,
		Username TEXT NOT NULL UNIQUE,
		Email TEXT NOT NULL UNIQUE,
		Password TEXT NOT NULL,
		Confirmed INTEGER DEFAULT 0
		
	);
	
	CREATE TABLE IF NOT EXISTS Category (
		CategoryID INTEGER PRIMARY KEY AUTOINCREMENT,
		CategoryName TEXT NOT NULL UNIQUE
	);
	
	CREATE TABLE IF NOT EXISTS Post (
		PostID INTEGER PRIMARY KEY AUTOINCREMENT,
		Title TEXT NOT NULL,
		Content TEXT NOT NULL,
		UserID INTEGER NOT NULL,
		CategoryID INTEGER,
		FOREIGN KEY (UserID) REFERENCES User(UserID) ON DELETE CASCADE,
		FOREIGN KEY (CategoryID) REFERENCES Category(CategoryID) ON DELETE SET NULL
	);
	
	CREATE TABLE IF NOT EXISTS Comment (
		CommentID INTEGER PRIMARY KEY AUTOINCREMENT,
		Content TEXT NOT NULL,
		PostID INTEGER NOT NULL,
		UserID INTEGER NOT NULL,
		FOREIGN KEY (PostID) REFERENCES Post(PostID) ON DELETE CASCADE,
		FOREIGN KEY (UserID) REFERENCES User(UserID) ON DELETE SET NULL
	);
	
	CREATE TABLE IF NOT EXISTS Like (
		LikeID INTEGER PRIMARY KEY AUTOINCREMENT,
		UserID INTEGER NOT NULL,
		PostID INTEGER,
		CommentID INTEGER,
		IsLike BOOLEAN NOT NULL,
		FOREIGN KEY (UserID) REFERENCES User(UserID) ON DELETE CASCADE,
		FOREIGN KEY (PostID) REFERENCES Post(PostID) ON DELETE CASCADE,
		FOREIGN KEY (CommentID) REFERENCES Comment(CommentID) ON DELETE CASCADE
	);
	`
	_, err = DB.Exec(schema)
	if err != nil {
		log.Fatal(err)
	}

	// Create indexes
	indexes := `
	CREATE INDEX IF NOT EXISTS idx_post_user ON Post(UserID);
	CREATE INDEX IF NOT EXISTS idx_post_category ON Post(CategoryID);
	CREATE INDEX IF NOT EXISTS idx_comment_post ON Comment(PostID);
	CREATE INDEX IF NOT EXISTS idx_comment_user ON Comment(UserID);
	CREATE INDEX IF NOT EXISTS idx_like_user ON Like(UserID);
	CREATE INDEX IF NOT EXISTS idx_like_post ON Like(PostID);
	CREATE INDEX IF NOT EXISTS idx_like_comment ON Like(CommentID);
	`
	_, err = DB.Exec(indexes)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Database schema and indexes created or already exist.")
}

// GetUserStats retrieves the number of posts and comments for a user
func GetUserStats(userID int) (numPosts, numComments int, err error) {
	// Count posts
	err = DB.QueryRow(`SELECT COUNT(*) FROM Post WHERE UserID = ?`, userID).Scan(&numPosts)
	if err != nil {
		return 0, 0, err
	}

	// Count comments
	err = DB.QueryRow(`SELECT COUNT(*) FROM Comment WHERE UserID = ?`, userID).Scan(&numComments)
	if err != nil {
		return 0, 0, err
	}

	return numPosts, numComments, nil
}
