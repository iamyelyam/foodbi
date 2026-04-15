import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'app.foodbi.kz',
  appName: 'FoodBI',
  webDir: 'dist',
  server: {
    // Let Capacitor use https so it trusts remote API
    androidScheme: 'https',
  },
  ios: {
    contentInset: 'always',
  },
}

export default config
