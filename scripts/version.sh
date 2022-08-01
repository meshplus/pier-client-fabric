GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ $GIT_BRANCH == "HEAD" ]; then
    echo $(git describe --tags HEAD)
else
    echo $GIT_BRANCH
fi