package main

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"gopkg.in/ini.v1"
)

func getSectionName(sec *ini.Section) string {
	_, commentText, _ := strings.Cut(sec.Comment, "#")
	for _, field := range strings.Fields(commentText) {
		fieldName, fieldValue, found := strings.Cut(field, "=")
		if found && fieldName == "name" {
			return fieldValue
		}
	}
	return ""
}

func setSectionName(sec *ini.Section, name string) {
	sec.Comment = fmt.Sprintf("name=%s", name)
}

func parseSingleIPNet(raw string) (*net.IPNet, error) {
	if raw == "" {
		return nil, errors.New("address is empty")
	}
	if strings.Contains(raw, ",") {
		return nil, errors.New("address contains multiple entries")
	}
	netIP, cidr, err := net.ParseCIDR(raw)
	if err != nil {
		return nil, err
	}
	cidr.IP = netIP
	return cidr, nil
}

func pruneEmptyKeys(section *ini.Section) {
	for k, v := range section.KeysHash() {
		if v == "" {
			section.DeleteKey(k)
		}
	}
}
