#!/bin/sh

while inotifywait -r -e modify --exclude "(.*.db.*|.*.swp|build)" .; do
	cd templates/
	qtc
	cd ../
	echo | nc -q 1 localhost 12345
done
