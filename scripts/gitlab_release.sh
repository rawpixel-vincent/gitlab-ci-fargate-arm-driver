#!/usr/bin/env bash

set -eo pipefail

tag=${1:-$CI_COMMIT_TAG}

if [[ -z "${tag}" ]]; then
    echo -e "\033[0;31m****** gitlab publishing disabled ******\033[0m"
    echo -e "usage:\n\t$0 tag"
    exit 0
fi

if [[ -z "${CI_JOB_TOKEN}" ]]; then
    echo -e "\033[0;31m****** Missing CI_JOB_TOKEN, cannot release ******\033[0m"
    exit 0
fi

api=${CI_API_V4_URL:-https://gitlab.com/api/v4}
projectID=${CI_PROJECT_ID:-16240616}

projectUrl=${CI_PROJECT_URL:-https://gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate}

changelogUrl="${projectUrl}/blob/${tag}/CHANGELOG.md"
s3=${CI_ENVIRONMENT_URL/\/index.html/}

release=$(cat <<EOS
{
  "name": "${tag}",
  "tag_name": "${tag}",
  "description": "See [the changelog](${changelogUrl}) :rocket:",
  "assets": {
    "links": [
      { "name": "linux amd64", "url": "$s3/fargate-linux-amd64" },
      { "name": "linux i386", "url": "$s3/fargate-linux-i386" },
      { "name": "others", "url": "$s3/index.html" }
    ]
  }
}
EOS
)

curl -f \
     --header 'Content-Type: application/json' \
     --header "JOB-TOKEN: ${CI_JOB_TOKEN}" \
     --data "${release}" \
     --request POST \
     "${api}/projects/${projectID}/releases"
