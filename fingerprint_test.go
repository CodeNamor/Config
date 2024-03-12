package config

import (
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestCreation(t *testing.T) {

	if HashCode() != "" {
		log.Error("hashcode should be empty")
	}

	NewHashCode("abc")
	if HashCode() != "abc" {
		log.Errorf("hashcode expected to be %v, got %v", "abc", HashCode())
	}
}

func TestHashCode(t *testing.T) {
	NewHashCode("abc")
	if HashCode() != "abc" {
		log.Errorf("hashcode should be %v", "abc")
	}

	NewHashCode("def")
	if HashCode() != "abc" {
		log.Errorf("hashcode should be %v", "abc")
	}
}
