git fetch --all
git merge origin/main

go build
chmod +x ./gitblog

systemctl restart gitblog
