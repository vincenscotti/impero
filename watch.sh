#!/bin/sh

while inotifywait -r -e modify --exclude "(.*.db.*|.*.swp|build)" .; do
	killall impero
	sleep 0.2
	cd templates/ &&
	qtc &&
	cd ../ &&
	go build &&
		(./impero -pass="" -debug=false &)
done
