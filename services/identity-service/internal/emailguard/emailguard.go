// Package emailguard rejects registrations coming from known disposable
// email providers. It is the MVP sybil-resistance layer: cheap to run,
// invisible to legitimate users, and catches the lazy majority of
// throwaway-account abuse. Stronger layers (SMS OTP, NIN verification)
// are deferred to later phases — see the roadmap.
package emailguard

import (
	"bufio"
	_ "embed"
	"strings"
)

//go:embed disposable_domains.txt
var rawList string

// Guard is a set of disposable email domains loaded once at process
// start. Zero-value is unusable — call NewGuard.
type Guard struct {
	blocked map[string]struct{}
}

// NewGuard parses the embedded blocklist. The list is small enough that
// a plain map is fine — no need for a trie or bloom filter at this
// scale. Refresh the source by copying a new snapshot on top of
// disposable_domains.txt.
func NewGuard() *Guard {
	set := make(map[string]struct{}, 512)
	scanner := bufio.NewScanner(strings.NewReader(rawList))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		set[strings.ToLower(line)] = struct{}{}
	}
	return &Guard{blocked: set}
}

// IsDisposable returns true if the domain portion of `email` (everything
// after the last @) matches the embedded blocklist. A malformed email
// with no @ is treated as not-disposable — the caller is responsible for
// validating email shape before this check.
func (g *Guard) IsDisposable(email string) bool {
	at := strings.LastIndexByte(email, '@')
	if at < 0 || at == len(email)-1 {
		return false
	}
	domain := strings.ToLower(strings.TrimSpace(email[at+1:]))
	_, ok := g.blocked[domain]
	return ok
}

// Size returns how many domains are in the blocklist. Handy for boot
// logs and tests — "blocklist loaded (N domains)".
func (g *Guard) Size() int { return len(g.blocked) }
