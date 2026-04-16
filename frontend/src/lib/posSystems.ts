// Catalog of POS systems we support (or plan to support).
// Add new entries here as integrations are built — UI dropdowns pull from this list.
//
// `id` is the value persisted to backend (locations.pos_system column).
// `label` is what users see in the dropdown.
// `enabled: false` rows are shown but greyed out — useful for "coming soon" markers.

export interface PosSystemOption {
  id: string
  label: string
  enabled: boolean
}

export const POS_SYSTEMS: PosSystemOption[] = [
  { id: 'iiko', label: 'iiko', enabled: true },
  { id: 'numier', label: 'NUMIER', enabled: true },
  { id: 'r_keeper', label: 'r_keeper', enabled: false },
  { id: 'poster', label: 'Poster', enabled: false },
  { id: 'manual', label: 'Manual', enabled: true },
]

export function findPosLabel(id: string): string {
  return POS_SYSTEMS.find((p) => p.id === id)?.label ?? id
}
