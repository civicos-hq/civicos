-- Add FOREIGN KEY constraints to every FK column, replacing GORM's implicit
-- ones that had no ON DELETE clause with explicit, policy-aware ones.
--
-- Deletion policy — the shape of a citizen-first record:
--   CASCADE  — the child cannot exist without the parent (upvotes,
--              signatures, followers, comments, org members, refresh
--              tokens, org children, in-org assignments/updates)
--   RESTRICT — the child represents authorship or ownership; deleting
--              the parent would destroy public record. Use anonymize/
--              soft-delete at the app layer instead (announcements,
--              issues/petitions/reps by their author, orgs/projects by
--              their creator).
--   SET NULL — the reference is optional and orphaning it is meaningful
--              (a user leaves a community; a project stops being tied to
--              a specific community).
--
-- Cross-service note: some constraints span service boundaries
-- (organization-service.issue_assignments.issue_id → community-service.
-- issues.id). This is deliberate for the MVP shared-DB architecture. If
-- and when services move to separate DBs, these become application-layer
-- checks — the CLAUDE.md service-boundaries note tracks this.
--
-- Idempotent-ish: drops any existing constraint of the same name first.

BEGIN;

-- ── Drop the three GORM-created implicit constraints so we can re-add them
--    with explicit ON DELETE behaviour.
ALTER TABLE issue_comments DROP CONSTRAINT IF EXISTS fk_issues_comments;
ALTER TABLE issues         DROP CONSTRAINT IF EXISTS fk_communities_issues;
ALTER TABLE petitions      DROP CONSTRAINT IF EXISTS fk_communities_petitions;

-- ── identity-service ────────────────────────────────────────────────────
ALTER TABLE refresh_tokens
  ADD CONSTRAINT fk_refresh_tokens_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE users
  ADD CONSTRAINT fk_users_community_id
    FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE SET NULL;

-- ── community-service ───────────────────────────────────────────────────
ALTER TABLE communities
  ADD CONSTRAINT fk_communities_created_by_id
    FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE issues
  ADD CONSTRAINT fk_issues_community_id
    FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE RESTRICT,
  ADD CONSTRAINT fk_issues_reported_by_id
    FOREIGN KEY (reported_by_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE issue_comments
  ADD CONSTRAINT fk_issue_comments_issue_id
    FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE,
  ADD CONSTRAINT fk_issue_comments_author_id
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE issue_upvotes
  ADD CONSTRAINT fk_issue_upvotes_issue_id
    FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE,
  ADD CONSTRAINT fk_issue_upvotes_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE petitions
  ADD CONSTRAINT fk_petitions_community_id
    FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE RESTRICT,
  ADD CONSTRAINT fk_petitions_created_by_id
    FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE petition_comments
  ADD CONSTRAINT fk_petition_comments_petition_id
    FOREIGN KEY (petition_id) REFERENCES petitions(id) ON DELETE CASCADE,
  ADD CONSTRAINT fk_petition_comments_author_id
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE petition_signatures
  ADD CONSTRAINT fk_petition_signatures_petition_id
    FOREIGN KEY (petition_id) REFERENCES petitions(id) ON DELETE CASCADE,
  ADD CONSTRAINT fk_petition_signatures_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE representatives
  ADD CONSTRAINT fk_representatives_community_id
    FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE RESTRICT,
  ADD CONSTRAINT fk_representatives_created_by_id
    FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE representative_comments
  ADD CONSTRAINT fk_representative_comments_representative_id
    FOREIGN KEY (representative_id) REFERENCES representatives(id) ON DELETE CASCADE,
  ADD CONSTRAINT fk_representative_comments_author_id
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE representative_followers
  ADD CONSTRAINT fk_representative_followers_representative_id
    FOREIGN KEY (representative_id) REFERENCES representatives(id) ON DELETE CASCADE,
  ADD CONSTRAINT fk_representative_followers_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE notifications
  ADD CONSTRAINT fk_notifications_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- ── organization-service ────────────────────────────────────────────────
ALTER TABLE organizations
  ADD CONSTRAINT fk_organizations_created_by_id
    FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE org_members
  ADD CONSTRAINT fk_org_members_organization_id
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
  ADD CONSTRAINT fk_org_members_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE announcements
  ADD CONSTRAINT fk_announcements_organization_id
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
  ADD CONSTRAINT fk_announcements_author_id
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE projects
  ADD CONSTRAINT fk_projects_organization_id
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
  ADD CONSTRAINT fk_projects_community_id
    FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE SET NULL,
  ADD CONSTRAINT fk_projects_created_by_id
    FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE issue_assignments
  ADD CONSTRAINT fk_issue_assignments_organization_id
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
  ADD CONSTRAINT fk_issue_assignments_issue_id
    FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE,
  ADD CONSTRAINT fk_issue_assignments_assigned_by_id
    FOREIGN KEY (assigned_by_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE progress_updates
  ADD CONSTRAINT fk_progress_updates_organization_id
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
  ADD CONSTRAINT fk_progress_updates_issue_id
    FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE,
  ADD CONSTRAINT fk_progress_updates_project_id
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
  ADD CONSTRAINT fk_progress_updates_author_id
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT;

COMMIT;
