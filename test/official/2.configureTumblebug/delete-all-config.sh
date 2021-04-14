#!/bin/bash

#function delete_ns() {
    FILE=../conf.env
    if [ ! -f "$FILE" ]; then
        echo "$FILE does not exist."
        exit
    fi

    TestSetFile=${4:-../testSet.env}
    
    FILE=$TestSetFile
    if [ ! -f "$FILE" ]; then
        echo "$FILE does not exist."
        exit
    fi
	source $TestSetFile
    source ../conf.env
    AUTH="Authorization: Basic $(echo -n $ApiUsername:$ApiPassword | base64)"

    echo "####################################################################"
    echo "## 0. Config: Delete ALL"
    echo "####################################################################"


    curl -H "${AUTH}" -sX DELETE http://$TumblebugServer/tumblebug/config | jq
    echo ""
#}

#delete_ns
