# Literary Lions Forum

A web forum for Literary Lions allowing users to communicate, associate categories with posts, like/dislike posts & comments, and filter posts.
Test


## Before starting the program

- Instal Golang

- Install SQLite:
```
sudo apt update
sudo apt install sqlite3
```

- HOW TO USE SQLite

Open SQLite Command Line Interface:
```
sqlite3 user.db
```

- Basic SQLite Commands:

View All Tables:
```
.tables
```
Describe Table Schema:
```
.schema tablename
```
Query Data:
```
SELECT * FROM user;
```
Exit SQLite CLI:
```
.exit
```

- Dockerize the Application

Build the Docker Image:
```
sudo docker build -t literary-lions .
```
Run the Docker Container:
```
sudo docker run -p 8080:8080 literary-lions
```

- HOW TO USE Docker

View Running Containers (split terminal window):
```
sudo docker ps
```
View Docker Logs:
```
sudo docker logs <container_id>
```


## Using SQLite Browser (optional)

Install SQLite Browser:
```
sudo add-apt-repository -y ppa:linuxgndu/sqlitebrowser
sudo apt update
sudo apt install sqlitebrowser
```
Open SQLite Browser:
```
sqlitebrowser
```


## starting the program

by entering 
```
go run .
```

a server will start at localhost:8080.
if you enter localhost:8080 in youre web browser you will enter our Literary Lions forum.


## Register

you can register by entering 
- Username
- Email
- Password
  
when you have registered you will get a confirmation email from literary.lions.verf@gmail.com.

here are 2 email you can test with

Email: lionsreviewer1@gmail.com \
Password: Reviewer1- \
Email: lionsreviewer2@gmail.com  
Password: Reviewer2test


## Login

you can now login using youre

- Email
- Password

if you dont remenber youre password you can click the reset password link below login.
then you can enter youre email and literary.lions.verf@gmail.com will send you a email with a reset password link.
click the link and you will get to a page where you can input youre new password


## Forum

if you are not logged in you can:
- filter by category
- read posts and comments 

if you are logged in:
- create posts
- comment on posts
- like/dislike post
- like/dislike comments
- filter by time/likes/category
- delete posts


## My Page / need to be logged in

here you can see youre:
- Username
- Email
- number of likes
- number of dislikes
- Number of Comments
- Number of posts
- delete accaunt


## My Posts / need to be logged in

here yoy can see all youre posts and likes and can click on them to get you to the post          


## Coders

Laura Levistö - Jonathan Dahl - 9/24
