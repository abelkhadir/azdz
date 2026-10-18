# social-network

## Run

With Docker:

```sh
docker compose up --build
```

Open `http://localhost:3000`. The frontend and backend run as separate
containers on ports 3000 and 8080. SQLite data and uploaded images use named
volumes, so they survive container restarts.

For local development, run `go run .` inside `backend`, then `npm install` and
`npm run dev` inside `frontend`.

Project:
```
Social Network
Objectives
You will have to create a Facebook-like social network that will contain the following features:

Followers
Profile
Posts
Groups
Notification
Chats
Instructions
Frontend
Frontend development is the art of creating sites and web applications that render on the "client-side". It includes everything that users experience directly: text colors and styles, images, graphs and tables, buttons, colors, and navigation menus. It focuses on making requests to the backend in order to get specific chunks of data to be used or send data to be stored on the backend.

HTML, CSS, and Javascript are the languages used for frontend development. Responsiveness and performance are two main objectives of the frontend. Frontend frameworks may be used to simplify a developer's work.

Framework
You will have to use a JS framework. It is up to you to choose which one you are going to use.

Frameworks will help you to organize and implement the features you want on your project, so that you can get more work done in an easier and faster way.

Some of the most known JS frameworks around are:

Next.js
Vue.js
Svelte
Mithril
Caution: Note that JS frameworks are different from JS libraries. JS libraries contain code snippets that are used to perform common JavaScript functions, while frameworks will help you by laying out the groundwork/building the bases for your JS project.

Backend
The backend is all of the technology required to process the incoming requests and generate and send responses to the client. This is typically divided into three major parts:

Server: this is the computer that receives requests. It acts as the entry point for all incoming requests. While there are specialized machines designed for this purpose, you can use your own computer as a server.

App: is the application running on the server that listens for requests, retrieves information from the database and sends responses. This is where the core logic of your social network resides. It contains the logic for handling various requests based on HTTP or other protocols. Some of these functions are known as middleware, which execute between receiving a request and sending a response.

Database: As you may already know, the database is used to organize and store data. Many requests sent to the server involve database queries. Clients may request information stored in the database or submit data to be added to it.

App
The backend may consist, like said above, of an app containing all the backend logic. This logic will therefore have several middleware, for example:

Authentication, since HTTP is a stateless protocol, we can use several ways to overcome and authenticate a client/user. You must use sessions and cookies.
Images handling, supporting various types of extensions. In this project you have to handle at least JPEG, PNG and GIF types. You will have to store the images, it can be done by storing the file/path in the database and saving the image in a specific file system.
Websocket, handling the connections in real time, between clients. This will help with the private chats.
For the web server you can take a look at caddy, it can serve your site, services and apps and its written in Go. Or you are free to create your own web server.

Sqlite
In order to store the data in your social network, you will use the database library SQLite. To structure your database and to achieve better performance we highly advise you to take a look at the entity relationship diagram and build one based on your own database.

To know more about SQLite you can check the SQLite page.

Migrate
You will have to create migrations for this project so every time the application runs, it creates the specific tables to make the project work properly.

For this, you must focus on a folder structure similar to this one:

student$ tree .
backend
├── pkg
│   ├── db
│   │   ├── migrations
│   │   │   └── sqlite
│   │   │       ├── 000001_create_users_table.down.sql
│   │   │       ├── 000001_create_users_table.up.sql
│   │   │       ├── 000002_create_posts_table.down.sql
│   │   │       └── 000002_create_posts_table.up.sql
│   │   └── sqlite
│   |       └── sqlite.go
|   |
|   └── ...other_pkgs.go
|
└── server.go
The folder structure is organized in a way that helps you to understand and use migrations, where you can apply it using a simple path, for example: file://backend/pkg/db/migrations/sqlite. It can be organized as you wish but do not forget that the application of migrations and the file organization will be tested.

For migrations you can use golang-migrate package or other package that better suits your project.

All migrations should be stored on a specific folder, as above. The sqlite.go should present the connection to the database, the applying of the migrations and other useful functionalities that you may need to implement.

This migration system can help you manage your time and testing, by filling your database.

docker
Containerizing the Backend and Frontend

Your project should consist of two Docker images, one for the backend and another for the frontend. Each of these containers serves a specific purpose and communicates with each other to provide a functional social network application.

Backend Container:

Create a Docker image for the backend of your social network application. This container will run the server-side logic of your application, handle requests from clients, and interact with the database.

Frontend Container:

Create a Docker image for the frontend of your social network application. This container will serve the client-side code, like HTML, CSS, and JavaScript files, to users browsers. It will also communicate with the backend via HTTP requests.

Tips:

Name your frontend Docker image appropriately.
Make sure that the backend container exposes the necessary ports for communication with the frontend and external clients and the frontend container exposes the appropriate port to serve the frontend content to users' browsers.
Authentication
In order for the users to use the social network they will have to make an account. So you will have to make a registration and login form. To register, every user should provide at least:

Email
Password
First Name
Last Name
Date of Birth
Avatar/Image (Optional)
Nickname (Optional)
About Me (Optional)
Note that the Avatar/Image, Nickname and About Me should be present in the form but the user can skip the filling of those fields.

When the user logins, he/she should stay logged in until he/she chooses a logout option that should be available at all times. For this you will have to implement sessions and cookies.

You can implement your own package for sessions and cookies or you can take a look at some packages to help you.

Followers
When navigating the social network the user should be able to follow and unfollow other users. Needless to say that to unfollow a user you have to be following him/her.

Regarding following someone, the user must initiate this by sending a follow request to the desired user. The recipient user can then choose to "accept" or "decline" the request. However, if the recipient user has a public profile (as explained in the next section), this request-and-accept process is bypassed and the user who sent the request automatically starts following the user with the pubic profile.

Profile
Every profile should contain :

User information (every information requested in the register form apart from the Password, obviously)
User activity
Every post made by the user
Followers and following users (display the users that are following the owner of the profile and who he/she is following)
There are two types of profiles: a public profile and a private profile. A public profile will display the information specified above to every user on the social network, while the private profile will only display that same information to their followers only.

When the user is in their own profile it should be available an option that allows the user to turn its profile public or private.

Posts
After a user is logged in he/she can create posts and comments on already created posts. While creating a post or a comment, the user can include an image or GIF.

The user must be able to specify the privacy of the post:

public (all users in the social network will be able to see the post)
almost private (only followers of the creator of the post will be able to see the post)
private (only the followers chosen by the creator of the post will be able to see it)
Groups
A user must be able to create a group. The group should have a title and a description given by the creator and he/she can invite other users to join the group.

The invited users need to accept the invitation to be part of the group. They can also invite other people once they are already part of the group. Another way to enter the group is to request to be in it and only the creator of the group would be allowed to accept or refuse the request.

To make a request to enter a group the user must find it first. This will be possible by having a section where you can browse through all groups.

When in a group, a user can create posts and comment the posts already created. These posts and comments will only be displayed to members of the group.

A user belonging to the group can also create an event, making it available for the other group users. An event should have:

Title
Description
Day/Time
2 Options (at least):
Going
Not going
After creating the event every user can choose one of the options for the event.

Chat
Users should be able to send private messages to other users that they are following or being followed, in other words, at least one of the users must be following the other.

When a user sends a message, the recipient will instantly receive it through Websockets if they are following the sender or if the recipient has a public profile.

It should be able for the users to send emojis to each other.

Groups should have a common chat room, so if a user is a member of the group he/she should be able to send and receive messages to this group chat.

Notifications
A user must be able to see the notifications in every page of the project. New notifications are different from new private messages and should be displayed in a different way!

A user should be notified if he/she:

has a private profile and some other user sends him/her a following request
receives a group invitation, so he can refuse or accept the request
is the creator of a group and another user requests to join the group, so he can refuse or accept the request
is member of a group and an event is created
Every other notification created by you that isn't on the list is welcomed too.

Allowed Packages
The standard Go packages are allowed
Gorilla websocket
golang-migrate
sql-migration
migration
sqlite3
bcrypt
gofrs/uuid or google/uuid
This project will help you learn about:

Authentication :
Sessions and cookies
Using and setting up Docker
Containerizing an application
Compatibility/Dependency
Creating images
SQL language
Manipulation of databases
Migrations
The basics of encryption
Websocket
```

Audit:
Functional
Has the requirement for the allowed packages been respected?
Open the project
When examining the file system of the backend, did you find a well-organized structure, similar to the example provided in the subject, with a clear separation of packages and migrations folders?
Is the file system for the frontend well organized?
Backend
Does the backend include a clear separation of responsibilities among its three major parts - Server, App, and Database?
Is there a server that effectively receives incoming requests and serves as the entry point for all requests to the application?
Does the application (App) running on the server effectively listen for requests, retrieve information from the database, and send responses?
Is the core logic of the social network implemented within the App component, including the logic for handling various types of requests based on HTTP or other protocols?
Database
Is SQLite being used in the project as the database?
Are clients able to request information stored in the database, and can they submit data to be added to it without encountering errors or issues?
Does the app implement a migration system?
Is that migration file system well organized? (like the example from the subject)
Start the social network application, then enter the database using the command "sqlite3 <database_name.db>".
Are the migrations being applied by the migration system?
Authentication
Does the app implement sessions for the authentication of the users?
Are the correct form elements being used in the registration? (Email, Password, First Name, Last Name, Date of Birth, Avatar/Image (Optional), Nickname (Optional), About Me (Optional))
Try to register a user.
During registration, when attempting to register a user, did the application correctly save the registered user to the database without any errors?
Try to log in with the user you just registered.
When attempting to log in with the user you just registered, did the login process work without any problems?
Try to log in with the user you created, but with a wrong password or email.
Did the application correctly detect and respond to the incorrect login details?
Try to register the same user you already registered.
Did the app detect if the email/user is already present in the database?
Open two browsers (ex: Chrome and Firefox), log in into one and refresh the other browsers.
Can you confirm that the browser non logged remains unregistered?
Using the two browsers, log in with different users in each one. Then refresh both the browsers.
Can you confirm that both browsers continue with the right users?
Followers
Try to follow a private user.
Are you able to send a following request to the private user?
Try to follow a public user.
Are you able to follow the public user without the need of sending a following request?
Open two browsers(ex: Chrome and Firefox), log in as two different private users and with one of them try to follow the other.
Is the user who received the request able to accept or decline the following request?
After following another user successfully try to unfollow him.
Were you able to do so?
Profile
Try opening your own profile.
Does the profile displays every information requested in the register form, apart from the password?
Try opening your own profile.
Does the profile displays every post created by the user?
Try opening your own profile.
Does the profile displays the users that you follow and the ones who are following you?
Try opening your own profile.
Are you able to change between private profile and public profile?
Open two browsers and log in with different users on them, with one of the users having a private profile and successfully follow that user.
Are you able to see a followed user private profile?
Using the two browsers with the same users, with one of the users having a private profile and be sure not to follow him.
Are you prevented from seeing a non-followed user private profile?
Using the two browsers with the users, with one of the users having a public profile and be sure not to follow him.
Are you able to see a non-followed user public profile?
Using the two browsers with the users, with one of the users having a public profile and successfully follow that user.
Are you able to see a followed user public profile?
Posts
Are you able to create a post and comment on already existing posts after logging in?
Try creating a post.
Are you able to include an image (JPG or PNG) or a GIF on it?
Try creating a comment.
Are you able to include an image (JPG or PNG) or a GIF on it?
Try creating a post.
Can you specify the type of privacy of the post (public, almost private, private)?
If you choose the private privacy option, can you specify the users that are allowed to see the post?
Groups
Try creating a group.
Were you able to invite one of your followers to join the group?
Open two browsers, log in with different users on each browser, follow each other and with one of the users create a group and invite the other user.
Did the other user received a group invitation that he/she can refuse/accept?
Using the same browsers and the same users, with one of the users create a group and with the other try to make a group entering request.
Did the owner of the group received a request that he/she can refuse/accept?
Can a user make group invitations, after being part of the group (being the user different from the creator of the group)?
Can a user make a group entering request (a request to enter a group)?
After being part of a group, can the user create posts and comment already created posts?
Try to create an event in a group.
Were you asked for a title, a description, a day/time and at least two options (going, not going)?
Using the same browsers and the same users, after both of them becoming part of the same group, create an event with one of them.
Is the other user able to see the event and vote in which option he wants?
Chat
Try and open two browsers (ex: Chrome and Firefox), log in with different users in each one. Then with one of the users try to send a private message to the other user.
Did the other user received the message in realtime?
Try and open two browsers (ex: Chrome and Firefox), log in with different users that are not following each other at all. Then with one of the users try to send a private message to the other user.
Can you confirm that it was not possible to create a chat between these two users?
Using the two browsers with the users start a chat between the two of them.
Did the chat between the users went well? (did not crash the server)
Try and open three browsers (ex: Chrome and Firefox or a private browser), log in with different users in each one. Then with one of the users try to send a private message to one of the other users.
Did only the targeted user received the message?
Using the three browsers with the users, enter with each user a common group. Then start sending messages to the common chat room using one of the users.
Did all the users that are common to the group receive the message in realtime?
Using the three browsers with the users, continue chatting between the users in the group.
Did the chat between the users went well? (did not crash the server)
Can you confirm that it is possible to send emojis via chat to other users?
Notifications
Can you check the notifications on every page of the project?
Open two browsers, log in as two different private users and with one of them try to follow the other.
Did the other user received a notification regarding the following request?
Open two browsers, log in with different users on each browser, follow each other and with one of the users create a group and invite the other user.
Did the invited user received a notification regarding the group invitation request?
Open two browsers, log in with different users on each browser, create a group with one of them and with the other send a group entering request.
Did the other user received a notification regarding the group entering request?
Open two browsers, log in with different users on each browser, become part of the same group with both users and with one of the users create an event.
Did the other user received a notification regarding the creation of the event?
Docker
Try to run the application and use the docker command "docker ps -a"
Can you confirm that there are two containers (backend and frontend), and both containers have non-zero sizes indicating that they are not empty?
Try to access the social network application through your web browser.
Were you able to access the social network application through your web browser after running the docker containers, confirming that the containers are running and serving the application as expected?
