while true; do
    # Check for changes in the working directory or index
    if ! git diff --quiet; then
        git add .
        git commit -m "updated $(date)"
        git push
    else
        echo "No changes to commit."
        sleep 15 # optional: wait 15 seconds before checking again
    fi
done
