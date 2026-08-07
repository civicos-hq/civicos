package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/civicos/identity-service/internal/domain"
	"github.com/civicos/identity-service/pkg/config"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type inMemoryUserStore struct {
	usersByID    map[string]*domain.User
	usersByEmail map[string]*domain.User
	repApps      map[string]*domain.RepresentativeApplication
	orgApps      map[string]*domain.OrganizationApplication
	// communities stands in for the communities table, which identity
	// reads directly from the shared database to validate a batch join.
	communities map[string]struct{}
}

func newInMemoryUserStore() *inMemoryUserStore {
	return &inMemoryUserStore{
		usersByID:    make(map[string]*domain.User),
		usersByEmail: make(map[string]*domain.User),
		repApps:      make(map[string]*domain.RepresentativeApplication),
		orgApps:      make(map[string]*domain.OrganizationApplication),
		communities:  make(map[string]struct{}),
	}
}

func (s *inMemoryUserStore) addCommunity(ids ...string) {
	for _, id := range ids {
		s.communities[id] = struct{}{}
	}
}

func (s *inMemoryUserStore) CountExistingCommunities(ids []string) (int64, error) {
	var n int64
	for _, id := range ids {
		if _, ok := s.communities[id]; ok {
			n++
		}
	}
	return n, nil
}

// JoinCommunities mirrors the real repository's rule: memberships are
// upserted, and the nominated primary is only written when the user has
// never had one.
func (s *inMemoryUserStore) JoinCommunities(userID string, communityIDs []string, primaryID string) error {
	user, ok := s.usersByID[userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	now := time.Now().UTC()
	for _, id := range communityIDs {
		exists := false
		for _, m := range user.Memberships {
			if m.CommunityID == id {
				exists = true
				break
			}
		}
		if exists {
			continue
		}
		user.Memberships = append(user.Memberships, domain.UserCommunityMembership{
			ID:          "membership-" + id,
			UserID:      userID,
			CommunityID: id,
			JoinedAt:    now,
		})
	}
	if user.PrimaryCommunityID == nil {
		p := primaryID
		user.PrimaryCommunityID = &p
	}
	active := primaryID
	user.ActiveCommunityID = &active
	return nil
}

func (s *inMemoryUserStore) FindByEmail(email string) (*domain.User, error) {
	user, ok := s.usersByEmail[email]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (s *inMemoryUserStore) FindByID(id string) (*domain.User, error) {
	user, ok := s.usersByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (s *inMemoryUserStore) CreateRegistration(
	user *domain.User,
	repApp *domain.RepresentativeApplication,
	orgApp *domain.OrganizationApplication,
) error {
	s.usersByID[user.ID] = user
	s.usersByEmail[user.Email] = user
	if repApp != nil {
		s.repApps[user.ID] = repApp
	}
	if orgApp != nil {
		s.orgApps[user.ID] = orgApp
	}
	return nil
}

func (s *inMemoryUserStore) CreateNotification(userID, title, body string, linkURL *string) error {
	return nil
}

func (s *inMemoryUserStore) JoinCommunity(userID, communityID string) error {
	user, ok := s.usersByID[userID]
	if !ok {
		return nil
	}
	user.ActiveCommunityID = &communityID
	// The first join claims primary but leaves the cooldown clock unset,
	// exactly as the real repository does — an initial assignment is not
	// a change.
	if user.PrimaryCommunityID == nil {
		p := communityID
		user.PrimaryCommunityID = &p
	}
	for _, membership := range user.Memberships {
		if membership.CommunityID == communityID {
			return nil
		}
	}
	user.Memberships = append(user.Memberships, domain.UserCommunityMembership{
		ID:          "membership-" + communityID,
		UserID:      userID,
		CommunityID: communityID,
		JoinedAt:    time.Now().UTC(),
	})
	return nil
}

func (s *inMemoryUserStore) SetActiveCommunity(userID, communityID string) error {
	user, ok := s.usersByID[userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	for _, membership := range user.Memberships {
		if membership.CommunityID == communityID {
			user.ActiveCommunityID = &communityID
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (s *inMemoryUserStore) SetPrimaryCommunity(userID, communityID string, changedAt time.Time) error {
	user, ok := s.usersByID[userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	user.PrimaryCommunityID = &communityID
	t := changedAt
	user.PrimaryCommunityChangedAt = &t
	return nil
}

func (s *inMemoryUserStore) UpdateProfile(userID, name, email string) error {
	user, ok := s.usersByID[userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	if name != "" {
		user.Name = name
	}
	if email != "" {
		delete(s.usersByEmail, user.Email)
		user.Email = email
		s.usersByEmail[email] = user
	}
	return nil
}

func (s *inMemoryUserStore) SetVerificationToken(userID, tokenHash string, expiresAt time.Time) error {
	user, ok := s.usersByID[userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	user.EmailVerificationTokenHash = &tokenHash
	user.EmailVerificationExpiresAt = &expiresAt
	return nil
}

func (s *inMemoryUserStore) FindByVerificationTokenHash(tokenHash string) (*domain.User, error) {
	for _, user := range s.usersByID {
		if user.EmailVerificationTokenHash != nil && *user.EmailVerificationTokenHash == tokenHash {
			return user, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *inMemoryUserStore) MarkVerified(userID string) error {
	user, ok := s.usersByID[userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	now := time.Now().UTC()
	user.EmailVerified = true
	user.EmailVerifiedAt = &now
	user.EmailVerificationTokenHash = nil
	user.EmailVerificationExpiresAt = nil
	return nil
}

func (s *inMemoryUserStore) SetPasswordResetToken(userID, tokenHash string, expiresAt time.Time) error {
	user, ok := s.usersByID[userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	user.PasswordResetTokenHash = &tokenHash
	user.PasswordResetExpiresAt = &expiresAt
	return nil
}

func (s *inMemoryUserStore) FindByPasswordResetTokenHash(tokenHash string) (*domain.User, error) {
	for _, user := range s.usersByID {
		if user.PasswordResetTokenHash != nil && *user.PasswordResetTokenHash == tokenHash {
			return user, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *inMemoryUserStore) ResetPassword(userID, newPasswordHash string) error {
	user, ok := s.usersByID[userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	user.PasswordHash = newPasswordHash
	user.PasswordResetTokenHash = nil
	user.PasswordResetExpiresAt = nil
	return nil
}

func (s *inMemoryUserStore) SoftDelete(userID string, reason *string) error {
	user, ok := s.usersByID[userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	// Anonymize by mirroring Repository.SoftDelete's semantics: email
	// becomes an unmatchable placeholder so re-registration under the
	// original address is unblocked, password hash is set to a value
	// no bcrypt.CompareHashAndPassword call can accept, and every
	// verification/reset token is invalidated.
	now := time.Now().UTC()
	delete(s.usersByEmail, user.Email)
	user.Email = "deleted-" + userID + "@civicos.deleted"
	user.Name = "[Deleted user]"
	user.PasswordHash = "x"
	user.AvatarURL = nil
	user.ActiveCommunityID = nil
	user.Memberships = nil
	user.EmailVerificationTokenHash = nil
	user.EmailVerificationExpiresAt = nil
	user.PasswordResetTokenHash = nil
	user.PasswordResetExpiresAt = nil
	user.DeletedAt = &now
	if reason != nil {
		user.DeletionReason = reason
	}
	s.usersByEmail[user.Email] = user
	return nil
}

// inMemoryRefreshStore lets rotation + replay tests run without Postgres.
// Keyed by token hash — mirrors the real repo's uniqueIndex.
type inMemoryRefreshStore struct {
	byID   map[string]*domain.RefreshToken
	byHash map[string]*domain.RefreshToken
}

func newInMemoryRefreshStore() *inMemoryRefreshStore {
	return &inMemoryRefreshStore{
		byID:   map[string]*domain.RefreshToken{},
		byHash: map[string]*domain.RefreshToken{},
	}
}

func (s *inMemoryRefreshStore) Create(t *domain.RefreshToken) error {
	if _, exists := s.byHash[t.TokenHash]; exists {
		return errors.New("duplicate token hash")
	}
	s.byID[t.ID] = t
	s.byHash[t.TokenHash] = t
	return nil
}

func (s *inMemoryRefreshStore) FindByHash(hash string) (*domain.RefreshToken, error) {
	if tok, ok := s.byHash[hash]; ok {
		return tok, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *inMemoryRefreshStore) Consume(id string, at time.Time) (int64, error) {
	tok, ok := s.byID[id]
	if !ok || tok.ConsumedAt != nil || tok.RevokedAt != nil {
		return 0, nil
	}
	tok.ConsumedAt = &at
	return 1, nil
}

func (s *inMemoryRefreshStore) RevokeFamily(familyID string, at time.Time) error {
	for _, tok := range s.byID {
		if tok.FamilyID == familyID && tok.RevokedAt == nil {
			revoked := at
			tok.RevokedAt = &revoked
		}
	}
	return nil
}

func (s *inMemoryRefreshStore) RevokeAllForUser(userID string, at time.Time) error {
	for _, tok := range s.byID {
		if tok.UserID == userID && tok.RevokedAt == nil {
			revoked := at
			tok.RevokedAt = &revoked
		}
	}
	return nil
}

// captureMailer records the last verification URL so VerifyEmail can be
// exercised without spinning up SMTP.
type captureMailer struct {
	lastTo      string
	lastSubject string
	lastText    string
}

func (m *captureMailer) Send(to, subject, htmlBody, textBody string) error {
	m.lastTo = to
	m.lastSubject = subject
	m.lastText = textBody
	return nil
}

func TestRegisterAndLoginFlow(t *testing.T) {
	cfg := &config.Config{JWTSecret: "12345678901234567890123456789012", AppURL: "http://localhost:5173"}
	repo := newInMemoryUserStore()
	svc := NewService(repo, newInMemoryRefreshStore(), cfg, &captureMailer{})

	user, regTokens, err := svc.Register(RegisterInput{
		Name:     "Ada",
		Email:    "ada@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Email != "ada@example.com" {
		t.Fatalf("expected registered email to be preserved, got %s", user.Email)
	}
	if user.EmailVerified {
		t.Fatalf("expected fresh registration to be unverified")
	}
	if regTokens == nil || regTokens.AccessToken == "" {
		t.Fatalf("expected register to auto-login by returning a token pair")
	}

	loggedInUser, tokens, err := svc.Login(LoginInput{
		Email:    "ada@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loggedInUser.ID != user.ID {
		t.Fatalf("expected logged in user id %s, got %s", user.ID, loggedInUser.ID)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("expected both tokens to be issued")
	}
}

// The disposable-email guard sits in front of Register and UpdateProfile
// as the MVP sybil-resistance layer. This test locks in that a
// well-known temp-mail domain (mailinator) is rejected with the
// documented error code, while a legit domain passes through.
func TestRegisterRejectsDisposableEmail(t *testing.T) {
	cfg := &config.Config{JWTSecret: "12345678901234567890123456789012", AppURL: "http://localhost:5173"}
	repo := newInMemoryUserStore()
	svc := NewService(repo, newInMemoryRefreshStore(), cfg, &captureMailer{})

	_, _, err := svc.Register(RegisterInput{
		Name:     "Sock Puppet",
		Email:    "throwaway@mailinator.com",
		Password: "password123",
	})
	if err == nil {
		t.Fatalf("expected disposable email to be rejected; got nil")
	}
	if err.Error() != "DISPOSABLE_EMAIL_DOMAIN" {
		t.Fatalf("expected DISPOSABLE_EMAIL_DOMAIN, got %q", err.Error())
	}

	// Sanity: legit domain still registers.
	if _, _, err := svc.Register(RegisterInput{
		Name:     "Real Person",
		Email:    "real@example.com",
		Password: "password123",
	}); err != nil {
		t.Fatalf("expected legit domain to pass through; got %v", err)
	}
}

// The check must also apply on email change — otherwise a citizen could
// sign up with gmail and then swap to a burner. Covers the parallel
// error path in UpdateProfile.
func TestUpdateProfileRejectsDisposableEmail(t *testing.T) {
	cfg := &config.Config{JWTSecret: "12345678901234567890123456789012", AppURL: "http://localhost:5173"}
	repo := newInMemoryUserStore()
	svc := NewService(repo, newInMemoryRefreshStore(), cfg, &captureMailer{})

	user, _, err := svc.Register(RegisterInput{
		Name:     "Ada",
		Email:    "ada@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("bootstrap register: %v", err)
	}

	burner := "burner@yopmail.com"
	_, err = svc.UpdateProfile(user.ID, UpdateProfileInput{Email: &burner})
	if err == nil {
		t.Fatalf("expected disposable email swap to be rejected; got nil")
	}
	if err.Error() != "DISPOSABLE_EMAIL_DOMAIN" {
		t.Fatalf("expected DISPOSABLE_EMAIL_DOMAIN, got %q", err.Error())
	}
}

func TestRepresentativeRegistrationCreatesPendingApplication(t *testing.T) {
	cfg := &config.Config{JWTSecret: "12345678901234567890123456789012", AppURL: "http://localhost:5173"}
	repo := newInMemoryUserStore()
	svc := NewService(repo, newInMemoryRefreshStore(), cfg, &captureMailer{})

	user, _, err := svc.Register(RegisterInput{
		Name:                 "Amina Yusuf",
		Email:                "amina@example.com",
		Password:             "password123",
		RequestedAccountType: "REPRESENTATIVE",
		Representative: &RepresentativeApplicationRegisterInput{
			FullName:     "Amina Yusuf",
			Title:        "Hon.",
			Position:     "Councillor",
			Constituency: "Ikeja Ward A",
			CommunityID:  "2d0fbf5e-6ccc-4f08-8395-d6cdbfb9b610",
		},
	})
	if err != nil {
		t.Fatalf("register representative: %v", err)
	}
	if user.Role != domain.RoleCitizen {
		t.Fatalf("expected applicant to remain citizen until approval, got %s", user.Role)
	}
	if user.RequestedAccountType != domain.AccountTypeRepresentative {
		t.Fatalf("expected requested account type representative, got %s", user.RequestedAccountType)
	}
	if user.ApprovalStatus != domain.ApprovalStatusPending {
		t.Fatalf("expected pending approval, got %s", user.ApprovalStatus)
	}
	if repo.repApps[user.ID] == nil {
		t.Fatalf("expected representative application to be persisted")
	}
}

func TestOrganizationRegistrationCreatesPendingApplication(t *testing.T) {
	cfg := &config.Config{JWTSecret: "12345678901234567890123456789012", AppURL: "http://localhost:5173"}
	repo := newInMemoryUserStore()
	svc := NewService(repo, newInMemoryRefreshStore(), cfg, &captureMailer{})

	user, _, err := svc.Register(RegisterInput{
		Name:                 "Tunde Adeyemi",
		Email:                "tunde@example.com",
		Password:             "password123",
		RequestedAccountType: "ORGANIZATION",
		Organization: &OrganizationApplicationRegisterInput{
			Name:         "Clean Lagos Initiative",
			Slug:         "clean-lagos-initiative",
			Kind:         "NGO",
			Jurisdiction: "STATE",
			State:        stringPtr("Lagos"),
		},
	})
	if err != nil {
		t.Fatalf("register organization: %v", err)
	}
	if user.Role != domain.RoleCitizen {
		t.Fatalf("expected applicant to remain citizen until approval, got %s", user.Role)
	}
	if user.RequestedAccountType != domain.AccountTypeOrganization {
		t.Fatalf("expected requested account type organization, got %s", user.RequestedAccountType)
	}
	if user.ApprovalStatus != domain.ApprovalStatusPending {
		t.Fatalf("expected pending approval, got %s", user.ApprovalStatus)
	}
	if repo.orgApps[user.ID] == nil {
		t.Fatalf("expected organization application to be persisted")
	}
}

func TestVerifyEmailFlow(t *testing.T) {
	cfg := &config.Config{JWTSecret: "12345678901234567890123456789012", AppURL: "http://localhost:5173"}
	repo := newInMemoryUserStore()
	mail := &captureMailer{}
	svc := NewService(repo, newInMemoryRefreshStore(), cfg, mail)

	if _, _, err := svc.Register(RegisterInput{
		Name:     "Ada",
		Email:    "ada@example.com",
		Password: "password123",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	rawToken := extractToken(t, mail.lastText)

	publicUser, tokens, err := svc.VerifyEmail(rawToken)
	if err != nil {
		t.Fatalf("verify-email: %v", err)
	}
	if !publicUser.EmailVerified {
		t.Fatalf("expected user to be verified after VerifyEmail")
	}
	if tokens == nil || tokens.AccessToken == "" {
		t.Fatalf("expected fresh tokens to be reissued so the client picks up emailVerified=true")
	}

	// Single-use: replaying the same token must now fail.
	if _, _, err := svc.VerifyEmail(rawToken); err == nil {
		t.Fatalf("expected replayed token to be rejected")
	}
}

func TestForgotAndResetPasswordFlow(t *testing.T) {
	cfg := &config.Config{JWTSecret: "12345678901234567890123456789012", AppURL: "http://localhost:5173"}
	repo := newInMemoryUserStore()
	mail := &captureMailer{}
	svc := NewService(repo, newInMemoryRefreshStore(), cfg, mail)

	if _, _, err := svc.Register(RegisterInput{
		Name:     "Ada",
		Email:    "ada@example.com",
		Password: "originalPassword",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Unknown email must return nil error (silent to prevent enumeration) and
	// must not send a mail.
	mail.lastText = ""
	if err := svc.RequestPasswordReset("nobody@example.com"); err != nil {
		t.Fatalf("expected nil error for unknown email, got %v", err)
	}
	if mail.lastText != "" {
		t.Fatalf("expected no mail sent for unknown email")
	}

	// Known email — must send a reset link.
	if err := svc.RequestPasswordReset("ada@example.com"); err != nil {
		t.Fatalf("forgot-password: %v", err)
	}
	if mail.lastSubject != "Reset your CivicOS password" {
		t.Fatalf("expected reset subject, got %q", mail.lastSubject)
	}
	rawToken := extractToken(t, mail.lastText)

	// Reset with a too-short password must be rejected before touching the DB.
	if _, _, err := svc.ResetPassword(rawToken, "short"); err == nil ||
		err.Error() != "PASSWORD_TOO_SHORT" {
		t.Fatalf("expected PASSWORD_TOO_SHORT, got %v", err)
	}

	publicUser, tokens, err := svc.ResetPassword(rawToken, "brandNewPassword")
	if err != nil {
		t.Fatalf("reset-password: %v", err)
	}
	if publicUser.Email != "ada@example.com" {
		t.Fatalf("expected same user, got %s", publicUser.Email)
	}
	if tokens == nil || tokens.AccessToken == "" {
		t.Fatalf("expected fresh tokens (auto-login) after reset")
	}

	// Old password must no longer log in; new one must.
	if _, _, err := svc.Login(LoginInput{Email: "ada@example.com", Password: "originalPassword"}); err == nil {
		t.Fatalf("expected old password to be rejected after reset")
	}
	if _, _, err := svc.Login(LoginInput{Email: "ada@example.com", Password: "brandNewPassword"}); err != nil {
		t.Fatalf("expected new password to work, got %v", err)
	}

	// Replaying the same reset token must fail (single-use).
	if _, _, err := svc.ResetPassword(rawToken, "anotherOne"); err == nil {
		t.Fatalf("expected replayed reset token to be rejected")
	}
}

func TestRefreshTokenRotation(t *testing.T) {
	cfg := &config.Config{JWTSecret: "12345678901234567890123456789012", AppURL: "http://localhost:5173"}
	repo := newInMemoryUserStore()
	refresh := newInMemoryRefreshStore()
	svc := NewService(repo, refresh, cfg, &captureMailer{})

	_, tokens, err := svc.Register(RegisterInput{
		Name: "Ada", Email: "ada@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	firstRefresh := tokens.RefreshToken
	if firstRefresh == "" {
		t.Fatalf("expected an initial refresh token")
	}

	// A successful rotation must return NEW tokens — the old refresh must no
	// longer work.
	rotated, err := svc.RefreshTokens(firstRefresh)
	if err != nil {
		t.Fatalf("first rotation: %v", err)
	}
	if rotated.RefreshToken == firstRefresh {
		t.Fatalf("rotation should hand out a fresh refresh token")
	}

	// Replay of the original consumed token: must (a) fail and (b) revoke
	// the whole family, so even the freshly-rotated token becomes unusable.
	if _, err := svc.RefreshTokens(firstRefresh); err == nil {
		t.Fatalf("replay of consumed refresh must be rejected")
	}
	if _, err := svc.RefreshTokens(rotated.RefreshToken); err == nil {
		t.Fatalf("family revocation must invalidate the rotated token too")
	}
}

func TestLogoutRevokesFamily(t *testing.T) {
	cfg := &config.Config{JWTSecret: "12345678901234567890123456789012", AppURL: "http://localhost:5173"}
	repo := newInMemoryUserStore()
	refresh := newInMemoryRefreshStore()
	svc := NewService(repo, refresh, cfg, &captureMailer{})

	_, tokens, err := svc.Register(RegisterInput{
		Name: "Ada", Email: "ada@example.com", Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// Rotate once so there are two tokens in the family — a live one and a
	// consumed one. Logout must invalidate both.
	rotated, err := svc.RefreshTokens(tokens.RefreshToken)
	if err != nil {
		t.Fatalf("rotate before logout: %v", err)
	}
	if err := svc.Logout(rotated.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.RefreshTokens(rotated.RefreshToken); err == nil {
		t.Fatalf("post-logout rotation must fail")
	}

	// Logout with an unknown refresh must NOT return an error — clients call
	// this from a stale session and expect to always succeed locally.
	if err := svc.Logout("does-not-exist"); err != nil {
		t.Fatalf("logout with unknown token should be a no-op, got %v", err)
	}
}

// ─── Batch join (onboarding) ──────────────────────────────────────────

const (
	uniAbuja    = "11111111-1111-4111-8111-111111111111"
	gwagwalada  = "22222222-2222-4222-8222-222222222222"
	nileUni     = "33333333-3333-4333-8333-333333333333"
	notACommuty = "44444444-4444-4444-8444-444444444444"
)

func newJoinFixture(t *testing.T) (*Service, *inMemoryUserStore, string) {
	t.Helper()
	cfg := &config.Config{JWTSecret: "12345678901234567890123456789012", AppURL: "http://localhost:5173"}
	repo := newInMemoryUserStore()
	repo.addCommunity(uniAbuja, gwagwalada, nileUni)
	svc := NewService(repo, newInMemoryRefreshStore(), cfg, &captureMailer{})

	user, _, err := svc.Register(RegisterInput{
		Name:     "Ada",
		Email:    "ada@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return svc, repo, user.ID
}

// The whole reason this endpoint exists: a student lives in one LGA and
// studies in another, and gets to say which of the two is home.
func TestJoinCommunitiesSetsChosenPrimary(t *testing.T) {
	svc, _, userID := newJoinFixture(t)

	user, err := svc.JoinCommunities(userID, []string{gwagwalada, uniAbuja}, uniAbuja)
	if err != nil {
		t.Fatalf("join communities: %v", err)
	}
	if len(user.Memberships) != 2 {
		t.Fatalf("expected 2 memberships, got %d", len(user.Memberships))
	}
	// Primary must be the nominated one, NOT the first in the list.
	if user.PrimaryCommunityID == nil || *user.PrimaryCommunityID != uniAbuja {
		t.Fatalf("expected primary %s, got %v", uniAbuja, user.PrimaryCommunityID)
	}
	if user.ActiveCommunityID == nil || *user.ActiveCommunityID != uniAbuja {
		t.Fatalf("expected active to follow primary, got %v", user.ActiveCommunityID)
	}
}

// Regression guard for the trap this endpoint was built to avoid. Joining
// communities one at a time stamps primary_community_changed_at on the first
// join, so nominating a different home immediately afterwards was refused
// with PRIMARY_COMMUNITY_COOLDOWN — a 30-day lockout earned during signup.
func TestJoinCommunitiesAvoidsDayOneCooldownLockout(t *testing.T) {
	svc, _, userID := newJoinFixture(t)

	// The old wizard's shape: join, join, and the primary is decided by
	// whichever request happened to land first, not by the user.
	if _, err := svc.JoinCommunity(userID, gwagwalada); err != nil {
		t.Fatalf("first join: %v", err)
	}
	if _, err := svc.JoinCommunity(userID, uniAbuja); err != nil {
		t.Fatalf("second join: %v", err)
	}
	me, err := svc.GetMe(userID)
	if err != nil {
		t.Fatalf("get me: %v", err)
	}
	if me.PrimaryCommunityID == nil || *me.PrimaryCommunityID != gwagwalada {
		t.Fatalf("sequential joins should leave primary at the first join, got %v", me.PrimaryCommunityID)
	}

	// The batch path does it in one write, so the user's choice lands.
	svc2, _, userID2 := newJoinFixture(t)
	user, err := svc2.JoinCommunities(userID2, []string{gwagwalada, uniAbuja}, uniAbuja)
	if err != nil {
		t.Fatalf("batch join: %v", err)
	}
	if user.PrimaryCommunityID == nil || *user.PrimaryCommunityID != uniAbuja {
		t.Fatalf("expected primary %s, got %v", uniAbuja, user.PrimaryCommunityID)
	}
}

// Picking a home during onboarding must not consume the 30-day allowance.
// It used to: both join paths stamped primary_community_changed_at on the
// initial assignment, so a citizen who chose wrong on their first day was
// locked out for a month before they had done anything at all.
func TestOnboardingChoiceLeavesTheCooldownUnspent(t *testing.T) {
	svc, _, userID := newJoinFixture(t)

	if _, err := svc.JoinCommunities(userID, []string{gwagwalada, uniAbuja}, uniAbuja); err != nil {
		t.Fatalf("batch join: %v", err)
	}

	// Immediately correcting the choice must be allowed.
	user, err := svc.SetPrimaryCommunity(userID, gwagwalada)
	if err != nil {
		t.Fatalf("first correction after onboarding must be free, got %v", err)
	}
	if user.PrimaryCommunityID == nil || *user.PrimaryCommunityID != gwagwalada {
		t.Fatalf("expected primary %s, got %v", gwagwalada, user.PrimaryCommunityID)
	}

	// But that correction does start the clock — a second change is not free.
	_, err = svc.SetPrimaryCommunity(userID, uniAbuja)
	var cooldown *PrimaryCommunityChangeError
	if !errors.As(err, &cooldown) {
		t.Fatalf("expected the second change to hit the cooldown, got %v", err)
	}
}

// Re-running the batch path must not become a cooldown bypass.
func TestJoinCommunitiesDoesNotOverwriteExistingPrimary(t *testing.T) {
	svc, _, userID := newJoinFixture(t)

	if _, err := svc.JoinCommunities(userID, []string{gwagwalada}, gwagwalada); err != nil {
		t.Fatalf("first batch: %v", err)
	}
	user, err := svc.JoinCommunities(userID, []string{nileUni}, nileUni)
	if err != nil {
		t.Fatalf("second batch: %v", err)
	}
	if user.PrimaryCommunityID == nil || *user.PrimaryCommunityID != gwagwalada {
		t.Fatalf("primary must stay %s, got %v", gwagwalada, user.PrimaryCommunityID)
	}
}

func TestJoinCommunitiesValidation(t *testing.T) {
	cases := []struct {
		name    string
		ids     []string
		primary string
		want    string
	}{
		{"empty selection", nil, uniAbuja, "COMMUNITY_SELECTION_REQUIRED"},
		{"primary outside selection", []string{gwagwalada}, uniAbuja, "PRIMARY_MUST_BE_SELECTED"},
		{"unknown community", []string{notACommuty}, notACommuty, "COMMUNITY_NOT_FOUND"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, userID := newJoinFixture(t)
			_, err := svc.JoinCommunities(userID, tc.ids, tc.primary)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("expected %s, got %v", tc.want, err)
			}
		})
	}

	t.Run("over the batch cap", func(t *testing.T) {
		svc, repo, userID := newJoinFixture(t)
		ids := make([]string, 0, MaxCommunitiesPerJoin+1)
		for i := 0; i <= MaxCommunitiesPerJoin; i++ {
			id := uuid.New().String()
			repo.addCommunity(id)
			ids = append(ids, id)
		}
		_, err := svc.JoinCommunities(userID, ids, ids[0])
		if err == nil || err.Error() != "TOO_MANY_COMMUNITIES" {
			t.Fatalf("expected TOO_MANY_COMMUNITIES, got %v", err)
		}
	})

	// Duplicates are the client's problem to make and ours to absorb: a
	// double-tapped checkbox must not trip the batch cap or double-write.
	t.Run("duplicates collapse", func(t *testing.T) {
		svc, _, userID := newJoinFixture(t)
		user, err := svc.JoinCommunities(userID, []string{uniAbuja, uniAbuja, gwagwalada}, uniAbuja)
		if err != nil {
			t.Fatalf("duplicate join: %v", err)
		}
		if len(user.Memberships) != 2 {
			t.Fatalf("expected 2 memberships after dedupe, got %d", len(user.Memberships))
		}
	})
}

// Prevent unused-import complaints in the rare test-only build case.
var _ = errors.New

func extractToken(t *testing.T, mailText string) string {
	t.Helper()
	const marker = "token="
	idx := strings.Index(mailText, marker)
	if idx < 0 {
		t.Fatalf("expected mail body to contain token, got: %q", mailText)
	}
	tail := mailText[idx+len(marker):]
	end := strings.IndexAny(tail, " \n\r")
	if end < 0 {
		return tail
	}
	return tail[:end]
}

func stringPtr(v string) *string { return &v }
