-- +goose Up
-- +goose StatementBegin

-- Explicit foreign keys with deliberate ON DELETE policies, replacing GORM's
-- implicit ones which carried no delete clause at all.
--
-- CASCADE where the child cannot exist without the parent; RESTRICT where the
-- child is authorship or ownership and deleting the parent would destroy
-- public record; SET NULL where the reference is optional and orphaning it
-- means something. The full reasoning is preserved in the file this replaces,
-- infrastructure/migrations/20260703_add_fk_constraints.sql.
--
-- Each constraint is guarded SEPARATELY on pg_constraint. Postgres has no
-- ADD CONSTRAINT IF NOT EXISTS, and these may already have been applied by
-- hand. Guarding a comma-joined ALTER as one unit would skip the constraints
-- that are missing whenever any sibling already exists — a silent partial
-- application, which is worse than either outcome.

DO $$
BEGIN
  ALTER TABLE issue_comments DROP CONSTRAINT IF EXISTS fk_issues_comments;

  ALTER TABLE issues DROP CONSTRAINT IF EXISTS fk_communities_issues;

  ALTER TABLE petitions DROP CONSTRAINT IF EXISTS fk_communities_petitions;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_refresh_tokens_user_id') THEN
    ALTER TABLE refresh_tokens ADD CONSTRAINT fk_refresh_tokens_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_users_community_id') THEN
    ALTER TABLE users ADD CONSTRAINT fk_users_community_id FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE SET NULL;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_communities_created_by_id') THEN
    ALTER TABLE communities ADD CONSTRAINT fk_communities_created_by_id FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_issues_community_id') THEN
    ALTER TABLE issues ADD CONSTRAINT fk_issues_community_id FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE RESTRICT;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_issues_reported_by_id') THEN
    ALTER TABLE issues ADD CONSTRAINT fk_issues_reported_by_id FOREIGN KEY (reported_by_id) REFERENCES users(id) ON DELETE RESTRICT;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_issue_comments_issue_id') THEN
    ALTER TABLE issue_comments ADD CONSTRAINT fk_issue_comments_issue_id FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_issue_comments_author_id') THEN
    ALTER TABLE issue_comments ADD CONSTRAINT fk_issue_comments_author_id FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_issue_upvotes_issue_id') THEN
    ALTER TABLE issue_upvotes ADD CONSTRAINT fk_issue_upvotes_issue_id FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_issue_upvotes_user_id') THEN
    ALTER TABLE issue_upvotes ADD CONSTRAINT fk_issue_upvotes_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_petitions_community_id') THEN
    ALTER TABLE petitions ADD CONSTRAINT fk_petitions_community_id FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE RESTRICT;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_petitions_created_by_id') THEN
    ALTER TABLE petitions ADD CONSTRAINT fk_petitions_created_by_id FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_petition_comments_petition_id') THEN
    ALTER TABLE petition_comments ADD CONSTRAINT fk_petition_comments_petition_id FOREIGN KEY (petition_id) REFERENCES petitions(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_petition_comments_author_id') THEN
    ALTER TABLE petition_comments ADD CONSTRAINT fk_petition_comments_author_id FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_petition_signatures_petition_id') THEN
    ALTER TABLE petition_signatures ADD CONSTRAINT fk_petition_signatures_petition_id FOREIGN KEY (petition_id) REFERENCES petitions(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_petition_signatures_user_id') THEN
    ALTER TABLE petition_signatures ADD CONSTRAINT fk_petition_signatures_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_representatives_community_id') THEN
    ALTER TABLE representatives ADD CONSTRAINT fk_representatives_community_id FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE RESTRICT;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_representatives_created_by_id') THEN
    ALTER TABLE representatives ADD CONSTRAINT fk_representatives_created_by_id FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_representative_comments_representative_id') THEN
    ALTER TABLE representative_comments ADD CONSTRAINT fk_representative_comments_representative_id FOREIGN KEY (representative_id) REFERENCES representatives(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_representative_comments_author_id') THEN
    ALTER TABLE representative_comments ADD CONSTRAINT fk_representative_comments_author_id FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_representative_followers_representative_id') THEN
    ALTER TABLE representative_followers ADD CONSTRAINT fk_representative_followers_representative_id FOREIGN KEY (representative_id) REFERENCES representatives(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_representative_followers_user_id') THEN
    ALTER TABLE representative_followers ADD CONSTRAINT fk_representative_followers_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_notifications_user_id') THEN
    ALTER TABLE notifications ADD CONSTRAINT fk_notifications_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_organizations_created_by_id') THEN
    ALTER TABLE organizations ADD CONSTRAINT fk_organizations_created_by_id FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_org_members_organization_id') THEN
    ALTER TABLE org_members ADD CONSTRAINT fk_org_members_organization_id FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_org_members_user_id') THEN
    ALTER TABLE org_members ADD CONSTRAINT fk_org_members_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_announcements_organization_id') THEN
    ALTER TABLE announcements ADD CONSTRAINT fk_announcements_organization_id FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_announcements_author_id') THEN
    ALTER TABLE announcements ADD CONSTRAINT fk_announcements_author_id FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_projects_organization_id') THEN
    ALTER TABLE projects ADD CONSTRAINT fk_projects_organization_id FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_projects_community_id') THEN
    ALTER TABLE projects ADD CONSTRAINT fk_projects_community_id FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE SET NULL;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_projects_created_by_id') THEN
    ALTER TABLE projects ADD CONSTRAINT fk_projects_created_by_id FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_issue_assignments_organization_id') THEN
    ALTER TABLE issue_assignments ADD CONSTRAINT fk_issue_assignments_organization_id FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_issue_assignments_issue_id') THEN
    ALTER TABLE issue_assignments ADD CONSTRAINT fk_issue_assignments_issue_id FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_issue_assignments_assigned_by_id') THEN
    ALTER TABLE issue_assignments ADD CONSTRAINT fk_issue_assignments_assigned_by_id FOREIGN KEY (assigned_by_id) REFERENCES users(id) ON DELETE RESTRICT;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_progress_updates_organization_id') THEN
    ALTER TABLE progress_updates ADD CONSTRAINT fk_progress_updates_organization_id FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_progress_updates_issue_id') THEN
    ALTER TABLE progress_updates ADD CONSTRAINT fk_progress_updates_issue_id FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_progress_updates_project_id') THEN
    ALTER TABLE progress_updates ADD CONSTRAINT fk_progress_updates_project_id FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_progress_updates_author_id') THEN
    ALTER TABLE progress_updates ADD CONSTRAINT fk_progress_updates_author_id FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE RESTRICT;
  END IF;
END $$;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Deliberately not reversible: dropping these restores a database where a
-- deleted user leaves donation and signature records pointing at nothing.
SELECT 1;
-- +goose StatementEnd
