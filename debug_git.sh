#!/bin/bash
git log -n 20 --graph --decorate > git_log_output.txt 2>&1
echo "Done" >> git_log_output.txt
