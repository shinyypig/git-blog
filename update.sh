git fetch --all

go build
chmod +x ./gitblog

systemctl restart gitblog
