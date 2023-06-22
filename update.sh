git fetch --all

/usr/local/go/bin/go build
chmod +x ./gitblog

systemctl restart gitblog
