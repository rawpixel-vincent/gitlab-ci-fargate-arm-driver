#!/usr/bin/env bash

set -eo pipefail

testsDir=${testsDir:-"./.tests/"}
goJunitReport=${goJunitReport:-"go-junit-report"}

function generate_coverage_report() {
	echo "Generating coverage HTML report"
	echo "To open execute:"
	echo "  x-www-browser ${testsDir}/coverage.html"

	go tool cover -o "${testsDir}/coverage.html" -html="${testsDir}/coverage.out"

	echo "Generating function report: ${testsDir}/coverage.txt"

	go tool cover -o "${testsDir}/coverage.txt" -func="${testsDir}/coverage.out"
	grep "^total:" "${testsDir}/coverage.txt"
}

function generate_junit_report() {
  local junitFile=${1}

  echo "Generating jUnit report: ${testsDir}/${junitFile}"

	${goJunitReport} < "${testsDir}/output.txt" > "${testsDir}/${junitFile}"
}

function normal() {
  local exitCode=0

  go test -v ./... -cover -covermode=count -coverprofile="${testsDir}/coverage.out.source" | tee "${testsDir}/output.txt" || exitCode=1
  grep -v -E "mock_.*\.go" "${testsDir}/coverage.out.source" > "${testsDir}/coverage.out"

  generate_coverage_report
  generate_junit_report "junit.xml"

  exit ${exitCode}
}

function race() {
  local exitCode=0

  go test -v -race ./... | tee "${testsDir}/output.txt" || exitCode=1

  generate_junit_report "junit-race.xml"

  exit ${exitCode}
}

case "${1}" in
  normal)
    normal
    ;;
  race)
    race
    ;;
  *)
    normal
    ;;
esac
