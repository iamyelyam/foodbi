import { create } from 'zustand'

interface AppState {
  activeLocationId: string | null
  setActiveLocation: (id: string | null) => void
}

export const useAppStore = create<AppState>((set) => ({
  activeLocationId: null,
  setActiveLocation: (id) => set({ activeLocationId: id }),
}))
