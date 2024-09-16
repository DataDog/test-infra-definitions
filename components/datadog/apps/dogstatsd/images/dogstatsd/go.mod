module dogstatsd

go 1.22

require github.com/DataDog/datadog-go/v5 v5.5.0

require (
	github.com/Microsoft/go-winio v0.5.0 // indirect
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007 // indirect
)

// Temporary replacement of the main branch until https://github.com/DataDog/datadog-go/pull/304 is released
replace github.com/DataDog/datadog-go/v5 => github.com/DataDog/datadog-go/v5 v5.5.1-0.20240327105053-fa1f6814eaf7
