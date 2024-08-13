-- Create tables
CREATE TABLE IF NOT EXISTS Category (
    CategoryID INTEGER PRIMARY KEY AUTOINCREMENT,
    CategoryName TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS User (
    UserID INTEGER PRIMARY KEY,
    Email TEXT UNIQUE NOT NULL,
    Username TEXT UNIQUE NOT NULL,
    Password TEXT NOT NULL
);


CREATE TABLE IF NOT EXISTS Post (
    PostID TEXT PRIMARY KEY,  -- Ensure this is TEXT if you're using a string for PostID
    Title TEXT,
    Content TEXT,
    UserID INTEGER,
    CategoryID INTEGER,
    FOREIGN KEY (UserID) REFERENCES User(UserID) ON DELETE SET NULL,  -- Adjust to ON DELETE SET NULL if needed
    FOREIGN KEY (CategoryID) REFERENCES Category(CategoryID)
);

CREATE TABLE IF NOT EXISTS Comment (
    CommentID INTEGER PRIMARY KEY AUTOINCREMENT,
    PostID TEXT NOT NULL,  -- Ensure this matches the type in Post table
    UserID INTEGER,  -- Adjust to allow NULL if using ON DELETE SET NULL
    Content TEXT NOT NULL,
    FOREIGN KEY (PostID) REFERENCES Post(PostID) ON DELETE CASCADE,
    FOREIGN KEY (UserID) REFERENCES User(UserID) ON DELETE SET NULL  -- Ensure UserID column allows NULL
);

CREATE TABLE IF NOT EXISTS LikesDislikes (
    ID INTEGER PRIMARY KEY AUTOINCREMENT,
    UserID INTEGER NOT NULL,
    PostID TEXT,  -- Ensure this matches the type in Post table
    CommentID INTEGER,
    IsLike BOOLEAN NOT NULL,
    FOREIGN KEY (UserID) REFERENCES User(UserID) ON DELETE CASCADE,
    FOREIGN KEY (PostID) REFERENCES Post(PostID) ON DELETE CASCADE,
    FOREIGN KEY (CommentID) REFERENCES Comment(CommentID) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS PasswordReset (
    Email TEXT NOT NULL,
    Token TEXT NOT NULL PRIMARY KEY,
    Expiry DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS Session (
    SessionID TEXT PRIMARY KEY,
    UserID INTEGER NOT NULL,
    CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (UserID) REFERENCES User(UserID)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_post_user ON Post(UserID);
CREATE INDEX IF NOT EXISTS idx_post_category ON Post(CategoryID);
CREATE INDEX IF NOT EXISTS idx_comment_post ON Comment(PostID);
CREATE INDEX IF NOT EXISTS idx_comment_user ON Comment(UserID);
CREATE INDEX IF NOT EXISTS idx_like_user ON LikesDislikes(UserID);
CREATE INDEX IF NOT EXISTS idx_like_post ON LikesDislikes(PostID);
CREATE INDEX IF NOT EXISTS idx_like_comment ON LikesDislikes(CommentID);


CREATE TABLE IF NOT EXISTS PostLikes (
    ID INTEGER PRIMARY KEY AUTOINCREMENT,
    UserID INTEGER NOT NULL,
    PostID TEXT,  -- Ensure this matches the type in Post table
    CommentID INTEGER,
    IsLike BOOLEAN NOT NULL,
    FOREIGN KEY (UserID) REFERENCES User(UserID) ON DELETE CASCADE,
    FOREIGN KEY (PostID) REFERENCES Post(PostID) ON DELETE CASCADE,
    FOREIGN KEY (CommentID) REFERENCES Comment(CommentID) ON DELETE CASCADE
);

-- To count likes and dislikes
SELECT 
    PostID, 
    SUM(CASE WHEN IsLike = TRUE THEN 1 ELSE 0 END) AS Likes, 
    SUM(CASE WHEN IsLike = FALSE THEN 1 ELSE 0 END) AS Dislikes 
FROM 
    PostLikes 
GROUP BY 
    PostID;