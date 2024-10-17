#!/bin/bash

TAG=${1}

if [ -z "$TAG" ]; then
    echo "Error: No version number provided."
    exit 1
fi
VERSION=${TAG##*v}

START_MARKER="<!--automq-operator release begin-->"
END_MARKER="<!--automq-operator release end-->"


echo "    \`\`\`shell
    wget -q https://github.com/cuisongliu/automq-operator/releases/download/v${VERSION}/automq-operator-v${VERSION}-sealos.tgz
    mkdir -p automq-operator && tar -zxvf automq-operator-v${VERSION}-sealos.tgz -C automq-operator
    cd automq-operator/deploy && bash install.sh
    \`\`\`" > replace_content.txt

awk -v start="$START_MARKER" -v end="$END_MARKER" -v newfile="replace_content.txt" '
BEGIN {printing=1}
$0 ~ start {print;system("cat " newfile);printing=0}
$0 ~ end {printing=1}
printing' README.md > temp.txt && mv temp.txt README.md
