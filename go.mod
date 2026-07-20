module github.com/code4ward/google-admanager-api-go

go 1.26

require github.com/hooklift/gowsdl v0.5.0

tool github.com/hooklift/gowsdl/cmd/gowsdl

require golang.org/x/sys v0.42.0 // indirect

require (
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	golang.org/x/oauth2 v0.36.0
)
