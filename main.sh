#!/bin/bash

while true; do
    echo "🔍 Checking for changes..."

    # Check if there are changes using `git status --porcelain`
    if [[ -z $(git status --porcelain) ]]; then
        echo "✅ No changes to commit."
        exit 0
    fi

    echo "➕ Adding changes..."
    git add .

    # Generate timestamped commit message
    timestamp=$(date)
    message="updated $timestamp"

    echo "📝 Committing changes..."
    if ! git commit -m "$message"; then
        echo "❌ Failed to commit changes."
        exit 1
    fi

    echo "🚀 Pushing to remote..."
    if ! git push; then
        echo "❌ Failed to push changes."
        exit 1
    fi

    echo "🎉 All changes pushed successfully."
done
