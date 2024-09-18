#!/bin/bash

git reset --hard
git pull origin dev

pm2 restart telex_be