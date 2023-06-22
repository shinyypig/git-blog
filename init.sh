#!/bin/bash

default_git_url="https://github.com/shinyypig/git-blog-extras.git"

read -p "Please input the extras git URL, the default is $default_git_url: " git_url

if [[ -z $git_url ]]; then
  git_url=$default_git_url
fi

echo "The extras git URL is: $git_url"

# 替换 gitblog.service 中的路径
current_path=$(pwd)
cp gitblog.service.bak gitblog.service
sed -i.bak "s|/root/git-blog|$current_path|g" gitblog.service
echo "current_path is $current_path"
cp gitblog.service /etc/systemd/system/ && systemctl daemon-reload


# 创建 tmp 文件夹，并克隆 git_url 到该文件夹
tmp_dir="./tmp"
mkdir -p $tmp_dir
git clone $git_url $tmp_dir
rm -rf $tmp_dir/.git

# 为每个文件夹在 ./git 文件夹中创建一个裸库，并将裸库的内容提取到 ./data 文件夹中
rm -rf ./git
rm -rf ./data

mkdir -p ./git
mkdir -p ./data
for dir in $tmp_dir/*; do
  if [ -d "$dir" ]; then
    base_dir=$(basename "$dir")
    mkdir "./git/$base_dir"
    cd "./git/$base_dir"
    git init --bare
    git symbolic-ref HEAD "refs/heads/main"

    # 初始化新的 git 存储库并提交
    cd "$current_path"
    cd "$dir"
    git init
    git add .
    git commit -m "init"
    git branch -m master main
    git checkout main
    git branch -D master
    echo "initilize git in $dir"
    
    # 设置新的裸库为远程，并推送
    git remote add origin "$current_path/git/$base_dir"
    git push origin main -f

    git clone "$current_path/git/$base_dir" "$current_path/data/$base_dir"

    cd "$current_path"
  fi
done

# 删除除了 _pages 文件夹之外的其他文件夹中的 .git 文件夹
for dir in ./data/*; do
  if [ -d "$dir" ] && [ "$dir" != "./data/_pages" ]; then
    rm -rf "$dir/.git"
  fi
done

# 删除 tmp 文件夹
rm -rf $tmp_dir

echo "Operation completed."
