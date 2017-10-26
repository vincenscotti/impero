package main

import "net/url"

type BLError struct {
	Message  string
	Redirect *url.URL
}
