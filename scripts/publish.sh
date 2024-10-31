#!/bin/bash

if [ -z ${1} ]; then
  echo "Usage: $0 <top-level-domain>"
  exit 1
fi

hut pages publish -d ${1} -p HTTPS site.tar.gz
hut pages publish -d www.${1} -p HTTPS site.tar.gz
# only publish gemini at top level domain
hut pages publish -d ${1} -p GEMINI site.tar.gz
