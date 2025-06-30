#!/bin/bash

while true; do
    echo "🔍 Checking for changes at $(date)..."

    # Check if there are any changes (unstaged or staged)
    if [[ -z $(git status --porcelain) ]]; then
        echo "✅ No changes to commit."
    else
        echo "➕ Adding changes..."
        git add .

        timestamp=$(date +"%Y-%m-%d %H:%M:%S")
        message="updated $timestamp"

        echo "📝 Committing changes..."
        if git commit -m "$message"; then
            echo "🚀 Pushing to remote..."
            if git push; then
                echo "🎉 All changes pushed successfully."
            else
                echo "❌ Failed to push changes."
            fi
        else
            echo "❌ Failed to commit changes."
        fi
    fi
    echo "⏳ Sleeping for seconds..."
    sleep 60
done
