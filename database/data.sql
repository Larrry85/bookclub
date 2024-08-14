-- data.sql

-- Insert initial data into Category table
INSERT INTO Category (CategoryName) 
VALUES ('New Category') 
ON CONFLICT(CategoryName) 
DO UPDATE SET CategoryName = excluded.CategoryName;

-- Insert initial data into User table
INSERT INTO User (UserID, Email, Username, Password) 
VALUES (1, 'user@example.com', 'username', 'hashed_password') 
ON CONFLICT(UserID) 
DO UPDATE SET Email = excluded.Email, Username = excluded.Username, Password = excluded.Password;

-- Insert initial data into User table with email conflict handling
INSERT INTO User (Email, Username, Password) 
VALUES ('user@example.com', 'username', 'hashed_password') 
ON CONFLICT(Email) 
DO UPDATE SET Username = excluded.Username, Password = excluded.Password;

-- Insert initial data into PostLikes table
INSERT INTO PostLikes (UserID, PostID, CommentID, IsLike) 
VALUES (1, 123, NULL, TRUE) 
ON CONFLICT(UserID, PostID, CommentID) 
DO UPDATE SET IsLike = excluded.IsLike;

-- Query to count likes and dislikes for each post
SELECT 
    PostID, 
    SUM(CASE WHEN IsLike = TRUE THEN 1 ELSE 0 END) AS Likes, -- Count of likes
    SUM(CASE WHEN IsLike = FALSE THEN 1 ELSE 0 END) AS Dislikes -- Count of dislikes
FROM 
    PostLikes 
GROUP BY 
    PostID; -- Group results by PostID

ALTER TABLE Post ADD COLUMN CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP;


CREATE TRIGGER set_created_at
AFTER INSERT ON Comment
FOR EACH ROW
BEGIN
    UPDATE Comment
    SET CreatedAt = datetime('now')
    WHERE rowid = NEW.rowid;
END;

SELECT c.Content, u.Username
FROM Comment c
JOIN User u ON c.UserID = u.UserID
WHERE c.PostID = ?
