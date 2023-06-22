git fetch --all
git rebase origin/main

export PATH=$PATH:/usr/local/go/bin
go build
chmod +x ./gitblog

systemctl restart gitblog
