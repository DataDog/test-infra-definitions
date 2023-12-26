module dogstatsd

go 1.20

require github.com/DataDog/datadog-go/v5 v5.3.0

require (
	github.com/Microsoft/go-winio v0.5.0 // indirect
	golang.org/x/sys v0.0.0-20210510120138-977fb7262007 // indirect
)

// This replace is necessary until https://github.com/DataDog/datadog-go/pull/291 and https://github.com/DataDog/datadog-go/pull/298
// are merged.
replace github.com/DataDog/datadog-go/v5 => github.com/DataDog/datadog-go/v5 v5.3.1-0.20231226143932-3c70095e0b5a
