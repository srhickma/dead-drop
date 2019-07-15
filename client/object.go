package main

import (
	"fmt"
	"strings"
)

const refSeparator = "#"

type ObjectReference struct {
	oid      string
	checksum string
}

func parseObjectReference(input string) (*ObjectReference, error) {
	split := strings.SplitN(input, refSeparator, 2)
	if len(split) != 2 {
		return nil, fmt.Errorf("malformed object reference")
	}

	or := &ObjectReference{
		oid: split[0],
		checksum: split[1],
	}
	return or, nil
}

func (or *ObjectReference) String() string {
	return fmt.Sprintf("%s%s%s", or.oid, refSeparator, or.checksum)
}
