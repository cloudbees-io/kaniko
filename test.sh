#!/bin/bash

TESTING_SHA=$(cat .cloudbees/testing/action.yml | sha1sum)

echo $TESTING_SHA