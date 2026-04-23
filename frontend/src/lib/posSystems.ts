// Catalog of POS systems we support (or plan to support).
// Add new entries here as integrations are built — UI dropdowns pull from this list.
//
// `id` is the value persisted to backend (locations.pos_system column).
// `label` is what users see in the dropdown.
// `enabled: false` rows are shown but greyed out — useful for "coming soon" markers.
// `hidden: true` rows are valid pos_system values but not shown in the picker
//   (e.g. auto-detected variants like iiko_cloud).

export interface PosSystemOption {
  id: string
  label: string
  enabled: boolean
  hidden?: boolean
}

export const POS_SYSTEMS: PosSystemOption[] = [
  { id: 'iiko', label: 'iiko', enabled: true },
  { id: 'iiko_cloud', label: 'iiko Cloud', enabled: true, hidden: true },
  { id: 'iikoweb', label: 'iikoWeb', enabled: true, hidden: true },
  { id: 'numier', label: 'NUMIER', enabled: true },
  { id: 'r_keeper', label: 'r_keeper', enabled: false },
  { id: 'poster', label: 'Poster', enabled: false },
  { id: 'manual', label: 'Manual', enabled: true },
]

export function findPosLabel(id: string): string {
  return POS_SYSTEMS.find((p) => p.id === id)?.label ?? id
}

// Detects whether an iiko URL points at a Cloud-hosted (iikoweb.ru) tenant
// or the public Cloud API endpoint. Returns true for Cloud, false otherwise.
// Server-API tenants typically live on *.iiko.it or custom domains and don't
// match either pattern.
export function isIikoCloudUrl(url: string): boolean {
  if (!url) return false
  const u = url.trim().toLowerCase()
  return /\.iikoweb\.ru/.test(u) || /api-ru\.iiko\.services/.test(u)
}
