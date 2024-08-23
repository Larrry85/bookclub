package post

import (
    "database/sql"
    "io"
    "net/http"
    "os"
    "time"
	"log"
    "github.com/google/uuid"
    "lions/session"
	"lions/database"
)

/*/ PostImage represents an image associated with a blog post.
type PostImage struct {
    ID        string       // Unique identifier for the post image
    PostID    string       // ID of the associated post
    UserID    int          // ID of the user who uploaded the image
    ImagePath string       // Path to the image file
    CreatedAt sql.NullTime // Timestamp when the image was uploaded
}*/

//
/* ImageData holds data for rendering the image upload page.
type ImageData struct {
    Authenticated bool   // Whether the user is authenticated
    Username      string // Username of the authenticated user
}*/

func uploadImage(w http.ResponseWriter, r *http.Request) {
    // Ensure the request method is POST
    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    // Parse the form data
    err := r.ParseMultipartForm(10 << 20) // 10 MB
    if err != nil {
        http.Error(w, "Unable to parse form", http.StatusBadRequest)
        return
    }

    // Retrieve the file from the form data
    file, handler, err := r.FormFile("image")
    if err != nil {
        http.Error(w, "Error retrieving the file", http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Generate a unique file name
    fileName := uuid.New().String() + "_" + handler.Filename
    filePath := "uploads/" + fileName

    // Create the file on the server
    dst, err := os.Create(filePath)
    if err != nil {
        http.Error(w, "Unable to create the file", http.StatusInternalServerError)
        return
    }
    defer dst.Close()

    // Copy the uploaded file to the server
    _, err = io.Copy(dst, file)
    if err != nil {
        http.Error(w, "Unable to save the file", http.StatusInternalServerError)
        return
    }

    // Retrieve the UserID from the session context
    userID, ok := r.Context().Value(session.UserID).(int)
    if !ok {
        http.Error(w, "Unable to retrieve user ID", http.StatusInternalServerError)
        return
    }

    // Save the image information to the database
    postImage := PostImage{
        ID:        uuid.New().String(),
        PostID:    r.FormValue("post_id"),
        UserID:    userID,  // Use the retrieved user ID here
        ImagePath: filePath,
        CreatedAt: sql.NullTime{Time: time.Now(), Valid: true},
    }

    err = savePostImageToDB(postImage)
    if err != nil {
        http.Error(w, "Unable to save image information", http.StatusInternalServerError)
        return
    }

	log.Println("Upload Image:", postImage)

    // Redirect to the post view page
    http.Redirect(w, r, "/post/view?id="+postImage.PostID, http.StatusSeeOther)
}



func savePostImageToDB(postImage PostImage) error {
    _, err := database.DB.Exec(`
        INSERT INTO PostImage (ImageID, PostID, UserID, ImagePath, CreatedAt)
        VALUES (?, ?, ?, ?, ?)`,
    	postImage.ID, postImage.PostID, postImage.UserID, postImage.ImagePath, postImage.CreatedAt)
		log.Println("Saving PostImage:", postImage)
		return err
}
