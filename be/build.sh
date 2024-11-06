#!/bin/bash

# 定义目标平台
platforms=("linux/amd64" "linux/arm64" "windows/amd64" "darwin/amd64" "darwin/arm64")

# 定义输出目录
output_dir="build"

rm -rf $output_dir

# 创建输出目录
mkdir -p $output_dir

# 循环编译每个平台的二进制文件
for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    output_name='docker-image-tools-'$GOOS'-'$GOARCH

    # Windows 平台需要添加 .exe 后缀
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi

    # 输出文件路径
    output_path="$output_dir/$output_name"

    echo "Building for $GOOS/$GOARCH..."
    env GOOS=$GOOS GOARCH=$GOARCH go build -o $output_path ./cmd/docker-image-tool

    # 检查构建是否成功
    if [ $? -ne 0 ]; then
        echo "Failed to build for $GOOS/$GOARCH"
    else
        echo "Successfully built $output_name"
    fi
done
