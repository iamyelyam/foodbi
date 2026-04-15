// Operational restaurant roles assigned to employees. Add new entries here as
// the org grows — UI dropdowns pull from this single source of truth.
//
// `id` is the value persisted to backend (users.role column).
// `label` is what users see in the picker.
//
// "Owner" grants full app permissions (gate for write actions across handlers
// keys off `role == "owner"`). Use sparingly — only when adding a co-founder
// or trusted partner who needs admin rights.
// Legacy 'employee' value is NOT in this list — pre-2026 rows kept it as default,
// but new employees pick a concrete operational role instead.

export interface EmployeeRoleOption {
  id: string
  label: string
}

export const EMPLOYEE_ROLES: EmployeeRoleOption[] = [
  { id: 'owner', label: 'Owner' },
  { id: 'general_manager', label: 'General Manager' },
  { id: 'manager', label: 'Manager' },
  { id: 'bartender', label: 'Bartender' },
  { id: 'waiter', label: 'Waiter' },
  { id: 'cashier', label: 'Cashier' },
  { id: 'accountant', label: 'Accountant' },
]

const LEGACY_LABELS: Record<string, string> = {
  owner: 'Owner',
  employee: 'Employee',
}

export function findRoleLabel(id: string): string {
  return (
    EMPLOYEE_ROLES.find((r) => r.id === id)?.label ??
    LEGACY_LABELS[id] ??
    id
  )
}
