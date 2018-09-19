#!/bin/sh

while inotifywait -r -e modify --exclude "(.*.db.*|.*.swp|build)" .; do
	killall impero
	sleep 0.2

	if [ $(basename `pwd`) != "templates" ]; then
		cd templates/
	fi
	qtc &&
	cd ../ &&
	go build &&
		(./impero -pass="" -debug=true &)
done
