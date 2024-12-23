-- Create tables

-- Create the Category table if it doesn't exist
CREATE TABLE IF NOT EXISTS Category (
    CategoryID INTEGER PRIMARY KEY AUTOINCREMENT,
    CategoryName TEXT NOT NULL UNIQUE
);

-- Insert categories if they do not already exist
INSERT OR IGNORE INTO Category (CategoryName) VALUES ('Adventure');
INSERT OR IGNORE INTO Category (CategoryName) VALUES ('Historical');
INSERT OR IGNORE INTO Category (CategoryName) VALUES ('Sci-Fi');
INSERT OR IGNORE INTO Category (CategoryName) VALUES ('General');
INSERT OR IGNORE INTO Category (CategoryName) VALUES ('Thriller');
INSERT OR IGNORE INTO Category (CategoryName) VALUES ('Fantasy');
INSERT OR IGNORE INTO Category (CategoryName) VALUES ('Science Fiction');
INSERT OR IGNORE INTO Category (CategoryName) VALUES ('Romance');
INSERT OR IGNORE INTO Category (CategoryName) VALUES ('Mystery');



-- Table to store user information
CREATE TABLE IF NOT EXISTS User (
    UserID INTEGER PRIMARY KEY AUTOINCREMENT, -- Unique identifier for each user
    Email TEXT UNIQUE NOT NULL, -- User's email address, must be unique
    Username TEXT UNIQUE NOT NULL, -- User's username, must be unique
    Password TEXT NOT NULL -- User's hashed password
);

-- Post Table
CREATE TABLE IF NOT EXISTS Post (
    PostID INTEGER PRIMARY KEY AUTOINCREMENT, -- Auto-increment unique identifier for each post
    Title TEXT NOT NULL, -- Title of the post
    Content TEXT NOT NULL, -- Content of the post
    UserID INTEGER, -- ID of the user who created the post
    CategoryID INTEGER, -- ID of the category to which the post belongs
    LastReplyUser INTEGER, -- ID of the user who last replied to the post (use UserID for relational integrity)
    LastReplyDate DATETIME, -- Date and time of the last reply
    CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP, -- Timestamp when the post was created
    LikesCount INTEGER DEFAULT 0,
    DislikesCount INTEGER DEFAULT 0,
    FOREIGN KEY (UserID) REFERENCES User(UserID) ON DELETE SET NULL, -- Foreign key to User table
    FOREIGN KEY (CategoryID) REFERENCES Category(CategoryID), -- Foreign key to Category table
    FOREIGN KEY (LastReplyUser) REFERENCES User(UserID) -- Foreign key to User table for LastReplyUser
);

CREATE TABLE IF NOT EXISTS PostImage (
    ID TEXT PRIMARY KEY, -- Unique identifier for each image (use TEXT for UUIDs)
    PostID INTEGER, -- Foreign key referencing the Post table
    UserID INTEGER, -- ID of the user who uploaded the image
    ImagePath TEXT, -- Path or URL to the image file
    CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP, -- Timestamp when the image was uploaded
    FOREIGN KEY (PostID) REFERENCES Post(PostID) ON DELETE CASCADE, -- Foreign key to Post table
    FOREIGN KEY (UserID) REFERENCES User(UserID) ON DELETE SET NULL -- Foreign key to User table
);


-- Table to store comments on posts
CREATE TABLE IF NOT EXISTS Comment (
    CommentID INTEGER PRIMARY KEY AUTOINCREMENT, -- Unique identifier for each comment
    PostID INTEGER, -- ID of the post to which the comment belongs
    UserID INTEGER, -- ID of the user who made the comment
    Content TEXT NOT NULL, -- Content of the comment
    CommentLikesCount INTEGER DEFAULT 0,
    CommentDislikesCount INTEGER DEFAULT 0,
    TaggedUser VARCHAR(255),
    CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP, -- Timestamp when the comment was created
    FOREIGN KEY (PostID) REFERENCES Post(PostID) ON DELETE CASCADE, -- Foreign key to Post table
    FOREIGN KEY (UserID) REFERENCES User(UserID) ON DELETE SET NULL -- Foreign key to User table
);

CREATE TABLE IF NOT EXISTS PostLikes (
    UserID INTEGER, -- ID of the user who liked/disliked
    PostID INTEGER, -- ID of the post being liked/disliked
    IsLike BOOLEAN, -- True for like, False for dislike
    PRIMARY KEY (UserID, PostID), -- Composite primary key: each user can like/dislike a post only once
    FOREIGN KEY (UserID) REFERENCES User(UserID),
    FOREIGN KEY (PostID) REFERENCES Post(PostID)
);

CREATE TABLE IF NOT EXISTS CommentLikes (
    UserID INTEGER, -- ID of the user who liked/disliked
    CommentID INTEGER, -- ID of the comment being liked/disliked
    IsLike BOOLEAN, -- True for like, False for dislike
    PRIMARY KEY (UserID, CommentID), -- Composite primary key: each user can like/dislike a comment only once
    FOREIGN KEY (UserID) REFERENCES User(UserID),
    FOREIGN KEY (CommentID) REFERENCES Comment(CommentID)
);

-- Table to store password reset tokens
CREATE TABLE IF NOT EXISTS PasswordReset (
    Email TEXT NOT NULL, -- Email of the user requesting a password reset
    Token TEXT NOT NULL PRIMARY KEY, -- Unique token for the password reset request
    Expiry DATETIME NOT NULL -- Expiry date and time of the token
);

-- Table to store user sessions
CREATE TABLE IF NOT EXISTS Session (
    SessionID TEXT PRIMARY KEY, -- Unique identifier for each session
    UserID INTEGER NOT NULL, -- ID of the user associated with the session
    CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP, -- Timestamp when the session was created
    FOREIGN KEY (UserID) REFERENCES User(UserID) -- Foreign key to User table
);

-- Create indexes to improve query performance
CREATE INDEX IF NOT EXISTS idx_post_user ON Post(UserID); -- Index on UserID in Post table
CREATE INDEX IF NOT EXISTS idx_post_category ON Post(CategoryID); -- Index on CategoryID in Post table
CREATE INDEX IF NOT EXISTS idx_comment_post ON Comment(PostID); -- Index on PostID in Comment table
CREATE INDEX IF NOT EXISTS idx_comment_user ON Comment(UserID); -- Index on UserID in Comment table
CREATE INDEX IF NOT EXISTS idx_like_user ON PostLikes(UserID); -- Index on UserID in PostLikes table
CREATE INDEX IF NOT EXISTS idx_like_post ON PostLikes(PostID); -- Index on PostID in PostLikes table
CREATE INDEX IF NOT EXISTS idx_like_comment ON CommentLikes(CommentID); -- Index on CommentID in CommentLikes table
CREATE INDEX IF NOT EXISTS idx_post_last_reply ON Post(LastReplyDate); -- Index on LastReplyDate in Post table
