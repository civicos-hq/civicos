import { useTranslation } from 'react-i18next';
import type {
  ApprovalStatus,
  IssueCategory,
  IssueStatus,
  NotificationType,
  PetitionStatus,
  RequestedAccountType,
  UserRole,
} from '@civicos/types';

/**
 * Ergonomic access to the shared `enums.*` translation namespace. Every
 * dashboard page renders enum values as badges/filters/labels, so bundling
 * the lookups here keeps the pages free of `t('enums.issueStatus.OPEN')`
 * strings sprinkled everywhere and gives us one place to change if we ever
 * add a new enum member.
 */
export function useEnumLabels() {
  const { t } = useTranslation();
  return {
    issueStatus: (v: IssueStatus | string) => t(`enums.issueStatus.${v}` as const),
    issueCategory: (v: IssueCategory | string) => t(`enums.issueCategory.${v}` as const),
    petitionStatus: (v: PetitionStatus | string) => t(`enums.petitionStatus.${v}` as const),
    userRole: (v: UserRole | string) => t(`enums.userRole.${v}` as const),
    requestedAccountType: (v: RequestedAccountType | string) =>
      t(`enums.requestedAccountType.${v}` as const),
    approvalStatus: (v: ApprovalStatus | string) => t(`enums.approvalStatus.${v}` as const),
    notificationType: (v: NotificationType | string) => t(`enums.notificationType.${v}` as const),
  };
}
