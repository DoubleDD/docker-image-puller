#!/bin/bash

# 定义输出目录
output_dir="build"

# 遍历输出目录中的所有文件
for filename in "$output_dir"/*; do
	# 检查文件是否存在
	if [ -f "$filename" ]; then
		base_filename=$(basename "$filename")
		echo ""
		echo "--------"
		echo "Compressing $base_filename"
		upx -9 -o "${output_dir}/upx-${base_filename}" "$filename"

		# 检查 upx 执行状态
		if [ $? -eq 0 ]; then
			echo "Compression successful, removing original file $base_filename"
			rm -f "$filename"
		else
			echo "Error: Compression of $base_filename failed."
		fi
	fi
done

echo ""
ls -hl $output_dir
rm -rf /Users/kedong/Downloads/docker/build/
cp -r $output_dir /Users/kedong/Downloads/docker/
