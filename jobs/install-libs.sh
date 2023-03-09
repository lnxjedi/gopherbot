#!/bin/bash -e

cd custom

if [ -e "requirements.txt" ]
then
    pip install -r requirements.txt
fi

if [ -e "Gemfile" ]
then
    bundle check || bundle install
fi