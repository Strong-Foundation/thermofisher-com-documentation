while true; do
    git add .
    git commit -m "updated $(date)"
    git push
done