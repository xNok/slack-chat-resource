#! /bin/bash
set -x

#! /bin/bash
type=$1
user=$2
request=$3

if [[ -z $type || -z $request ]]; then
    echo "Required arguments: <resource type> <request file>"
    exit 1
fi

cat "$request" | docker run --rm -i \
-e BUILD_NAME=mybuild \
-e BUILD_JOB_NAME=myjob \
-e BUILD_PIPELINE_NAME=mypipe \
-e BUILD_TEAM_NAME=myteam \
-e ATC_EXTERNAL_URL="https://example.com" \
-v "C:/Users/dlj5826/Documents/Projects/DD/slack-chat-resource/$type/out:/tmp/resource" $user/slack-$type-resource /opt/resource/out /tmp/resource
