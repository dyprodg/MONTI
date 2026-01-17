import { UserManager, User as OidcUser, WebStorageStateStore } from 'oidc-client-ts'
import { Location } from '../types'

// Business Unit to Location mapping
const BU_LOCATION_MAPPING: Record<string, Location[]> = {
  SGB: ['munich', 'frankfurt'],
  NGB: ['berlin', 'hamburg'],
  RGB: ['remote'],
}

const ALL_LOCATIONS: Location[] = ['berlin', 'munich', 'hamburg', 'frankfurt', 'remote']

export interface User {
  email: string
  name: string
  role: string
  groups: string[]
  businessUnits: string[]
  allowedLocations: Location[]
}

class AuthService {
  private userManager: UserManager

  constructor() {
    const config = {
      authority: import.meta.env.VITE_OIDC_ISSUER || 'http://localhost:8180/realms/monti',
      client_id: import.meta.env.VITE_OIDC_CLIENT_ID || 'monti-app',
      redirect_uri: import.meta.env.VITE_OIDC_REDIRECT_URI || 'http://localhost:5173/callback',
      post_logout_redirect_uri: window.location.origin,
      response_type: 'code',
      scope: 'openid profile email',
      userStore: new WebStorageStateStore({ store: window.localStorage }),
      automaticSilentRenew: true,
      silent_redirect_uri: `${window.location.origin}/silent-renew.html`,
    }

    this.userManager = new UserManager(config)

    // Handle silent renew errors
    this.userManager.events.addSilentRenewError((error) => {
      console.error('Silent renew error:', error)
    })
  }

  // Start login flow
  async login(): Promise<void> {
    await this.userManager.signinRedirect()
  }

  // Handle callback after login
  async handleCallback(): Promise<User | null> {
    try {
      const user = await this.userManager.signinRedirectCallback()
      return this.mapUser(user)
    } catch (error) {
      console.error('Login callback error:', error)
      return null
    }
  }

  // Get current user
  async getUser(): Promise<User | null> {
    const oidcUser = await this.userManager.getUser()
    if (!oidcUser || oidcUser.expired) {
      return null
    }
    return this.mapUser(oidcUser)
  }

  // Get access token
  async getToken(): Promise<string | null> {
    const user = await this.userManager.getUser()
    return user?.access_token || null
  }

  // Logout
  async logout(): Promise<void> {
    await this.userManager.signoutRedirect()
  }

  // Check if user is authenticated
  async isAuthenticated(): Promise<boolean> {
    const user = await this.userManager.getUser()
    return user !== null && !user.expired
  }

  // Map OIDC user to our User type
  private mapUser(oidcUser: OidcUser): User {
    const profile = oidcUser.profile
    const groups = this.extractGroups(profile)
    const role = this.extractRole(profile)
    const businessUnits = this.extractBusinessUnits(groups)
    const allowedLocations = this.computeAllowedLocations(role, businessUnits)

    return {
      email: (profile.email as string) || 'unknown@monti.local',
      name: (profile.name as string) || (profile.preferred_username as string) || 'Unknown User',
      role,
      groups,
      businessUnits,
      allowedLocations,
    }
  }

  // Extract role from token claims
  private extractRole(profile: any): string {
    // Check different possible locations for roles
    // Keycloak puts roles in realm_access.roles
    if (profile.realm_access?.roles) {
      const roles = profile.realm_access.roles
      if (roles.includes('admin')) return 'admin'
      if (roles.includes('manager')) return 'manager'
      if (roles.includes('viewer')) return 'viewer'
    }

    // AWS Cognito puts roles in cognito:groups
    if (profile['cognito:groups']) {
      const groups = profile['cognito:groups']
      if (groups.includes('monti-admins')) return 'admin'
      if (groups.includes('monti-managers')) return 'manager'
      if (groups.includes('monti-viewers')) return 'viewer'
    }

    // Check custom groups attribute
    if (profile['custom:groups']) {
      const groups = profile['custom:groups']
      if (groups.includes('monti-admins')) return 'admin'
      if (groups.includes('monti-managers')) return 'manager'
      if (groups.includes('monti-viewers')) return 'viewer'
    }

    // Default to viewer
    return 'viewer'
  }

  // Extract groups from token claims
  private extractGroups(profile: any): string[] {
    return profile.groups || profile['cognito:groups'] || profile['custom:groups'] || []
  }

  // Extract business units from groups
  // Groups are expected in format: /business-units/SGB, /business-units/NGB, etc.
  private extractBusinessUnits(groups: string[]): string[] {
    const buPrefix = '/business-units/'
    const businessUnits: string[] = []

    for (const group of groups) {
      if (group.startsWith(buPrefix)) {
        let bu = group.substring(buPrefix.length)
        // Remove any trailing path components
        const slashIndex = bu.indexOf('/')
        if (slashIndex > 0) {
          bu = bu.substring(0, slashIndex)
        }
        if (bu) {
          businessUnits.push(bu)
        }
      }
    }

    return businessUnits
  }

  // Compute allowed locations from role and business units
  // Admin role gets all locations; otherwise, locations are derived from BUs
  private computeAllowedLocations(role: string, businessUnits: string[]): Location[] {
    // Admin role sees everything
    if (role === 'admin') {
      return ALL_LOCATIONS
    }

    // Build unique set of allowed locations from all assigned BUs
    const locationSet = new Set<Location>()
    for (const bu of businessUnits) {
      const locations = BU_LOCATION_MAPPING[bu]
      if (locations) {
        for (const loc of locations) {
          locationSet.add(loc)
        }
      }
    }

    return Array.from(locationSet)
  }
}

export const authService = new AuthService()
