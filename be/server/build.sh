#!/bin/bash

# 定义目标平台
platforms=(
	 "linux/amd64"
	"linux/arm64"
)

# 定义输出目录
output_dir="build"

rm -rf $output_dir

# 创建输出目录
mkdir -p $output_dir

# 循环编译每个平台的二进制文件
for platform in "${platforms[@]}"; do
	platform_split=(${platform//\// })
	GOOS=${platform_split[0]}
	GOARCH=${platform_split[1]}
	output_name='docker-tools-'$GOOS'-'$GOARCH


	# 输出文件路径
	output_path="$output_dir/$output_name"

	echo "Building for $GOOS/$GOARCH..."
	env GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w" -o $output_path ./cmd/docker-image-tool

	# 检查构建是否成功
	if [ $? -ne 0 ]; then
		echo "Failed to build for $GOOS/$GOARCH"
	else
		echo "Successfully built $output_name"
		chmod +x $output_path
	fi
done

echo ""
ls -hl build
rm -rf /Users/kedong/Downloads/docker/build/
cp -r build /Users/kedong/Downloads/docker/
