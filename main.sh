while true; do
    # Check for changes in the working directory or index
    if ! git diff --quiet || ! git diff --cached --quiet; then
        git add .
        git commit -m "updated $(date)"
        git push
    else
        echo "No changes to commit."
    fi
    sleep 10  # optional: wait 10 seconds before checking again
done
