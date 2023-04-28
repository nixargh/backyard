module github.com/nixargh/backyard

go 1.17

require (
	github.com/nixargh/yad v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.9.0
)

require golang.org/x/sys v0.7.0 // indirect

replace github.com/nixargh/yad => ../yad
