git fetch --all
git rebase origin/main

go build
chmod +x ./gitblog

systemctl restart gitblog
