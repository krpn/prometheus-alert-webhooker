#!/bin/bash

# Builds the image then pushes it to the Github apckage manager
# Note if no token is provided, the script will be able to build the image but won't push it to github
set -ex

# Defaults
version="$(date '+%Y%m%d%H%M%S')";
user="";
token="dummy";

while [[ $# -gt 0 ]] ;
do
    opt="$1";
    shift;              #expose next argument
    case "$opt" in
        "--version" )
           version="$1"; shift;;
        "--user" )
           user="$1"; shift;;
        "--token" )
           token="$1"; shift;;
        *) echo >&2 "Invalid option: $@"; exit 1;;
   esac
done

echo "Running command with user: $user, version: $version"

if [ "$version" = "" ]; then
    echo "Image version is required (e.g. --version 12345)."
    exit 1
fi

if [ "$user" = "" ]; then
    echo "User name is required to access GitHub packages (e.g. --user myuser)."
    exit 1
fi

if [ "$token" = "" ]; then
    echo "Docker registry access token for user $user is required (e.g. --token 12345)."
    exit 1
fi

name="${user}/prometheus-alert-webhooker"

# Builds the artefact
docker build -t $name:$version .
# Also tag as latest
docker tag $name:$version $name:latest

if [[ ${token} = "dummy" ]]
then
  echo "Invalid token found... exit without pushing the image to the registry."
  exit 0
fi

# login to the docker repo, the token should be on an environment variable
echo $token | docker login -u $user --password-stdin

docker push $name:$version
docker push $name:latest

echo "Finished!"
