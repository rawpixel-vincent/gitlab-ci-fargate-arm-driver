// +build tools

package fargate

// These imports are to force `go mod tidy` not to remove that tools we depend
// on development. This is explained in great detail in
// https://marcofranssen.nl/manage-go-tools-via-go-modules/
import (
	_ "github.com/jstemmer/go-junit-report" // converting go test results to jUnit report file
	_ "github.com/mitchellh/gox"            // cross-compilation of the binary
)
