#!/bin/bash

mkdir -p daily

DATE=$(date +"%Y-%m-%d")
FILENAME="daily/$DATE.md"
EDITOR=$(git config core.editor)
if [ -f "$FILENAME" ]; then
    $EDITOR "$FILENAME" &
    echo "File $FILENAME already exists. Exiting."
    exit 1
fi

cat << EOF > "$FILENAME"
# $DATE

EOF

$EDITOR "$FILENAME" &

echo "Created $FILENAME"