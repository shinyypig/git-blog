# Git Blog

![license](https://img.shields.io/github/license/shinyypig/git-blog)
![last_commit](https://img.shields.io/github/last-commit/shinyypig/git-blog)
![downloads](https://img.shields.io/github/downloads/shinyypig/git-blog/total)

Git Blog is a lightweight blog system based on Git and Markdown.

Features:

-   Lightweight, no database required, cost only 8.1MB memory.
-   Git based, you can use git to manage your blog.
-   Markdown only, you can write your blog in Markdown.
-   Repository compatible , a repository can be used as a blog.
-   LaTeX support, you can write LaTeX in Markdown.
-   Light/Dark theme, switch between light and dark theme automatically.
-   Highly customizable, you can customize the theme and the template as you want.

Vist my website: [https://shinyypig.top](https://shinyypig.top) to see the demo.

## Installation

Clone this repository to your server:

```bash
git clone https://github.com/shinyypig/git-blog.git && cd git-blog
```

If you are familiar with linux, you can download [go](https://go.dev/doc/install) and build `gitblog` yourself:

```bash
go build
```

Or you can use the `build.sh` script to build it (only tested in debain):

```bash
sh build.sh
```

If success, you will get a `gitblog` executable file in the folder. You can run it directly:

```bash
./gitblog
```

If you want to install it as a service, you can use the following command:

```bash
cp gitblog.service /etc/systemd/system/ && systemctl daemon-reload
```

Before you copy the service file, you should edit it to make sure the `WorkingDirectory` and `ExecStart` are correct in `gitblog.service`. The default file looks like this:

```bash
[Unit]
Description=GitBlog Server

[Service]
ExecStart=/root/git-blog/gitblog
WorkingDirectory=/root/git-blog
Restart=always

[Install]
WantedBy=multi-user.target
```

Use the following command to play with the service:

```bash
systemctl start gitblog
systemctl stop gitblog
systemctl restart gitblog
systemctl status gitblog
```

If you want it to start automatically when the system starts, you can use the following command:

```bash
systemctl enable gitblog
```

## Usage

Git Blog serves as a git server, in which every post is a git repository, so you can use git to manage your post.

You can use the following command to download a post:

```bash
git clone http://yourdomain.com/yourpost.git
```

If it does not exist, then the server will create a new post for you.

You can modify the post in your local machine, and use `git push` to push it to the server.

Note that Git Blog will render the `README.md` file in the repository as the post, hence, your repositories on other git servers are also compatible with Git Blog. Simply push your repositories to the server, and it will be rendered as a post.

For more information, visit the **welcome** page after you run the server.

## Things to do

-   Add support for mobile devices.
-   Maybe a web editor?
