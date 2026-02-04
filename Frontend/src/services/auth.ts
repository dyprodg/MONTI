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
      monitorSession: false,
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

  // Decode the payload of a JWT without verification (we only need claims, the
  // backend verifies the signature). This lets us read realm_access from the
  // access token, which Keycloak always populates — unlike the ID token.
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private decodeJwtPayload(token: string): any {
    try {
      const parts = token.split('.')
      if (parts.length !== 3) return null
      const payload = parts[1].replace(/-/g, '+').replace(/_/g, '/')
      return JSON.parse(atob(payload))
    } catch {
      return null
    }
  }

  // Map OIDC user to our User type
  private mapUser(oidcUser: OidcUser): User {
    const profile = oidcUser.profile
    const accessClaims = this.decodeJwtPayload(oidcUser.access_token)
    const groups = this.extractGroups(profile, accessClaims)
    const role = this.extractRole(profile, accessClaims)
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

  // Extract role from token claims — checks access token first (always has
  // realm_access), then falls back to ID token profile.
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private extractRole(profile: any, accessClaims: any): string {
    // Check access token realm_access.roles (Keycloak always includes this)
    const realmRoles = accessClaims?.realm_access?.roles || profile.realm_access?.roles
    if (realmRoles && Array.isArray(realmRoles)) {
      if (realmRoles.includes('admin')) return 'admin'
      if (realmRoles.includes('supervisor')) return 'supervisor'
      if (realmRoles.includes('manager')) return 'manager'
      if (realmRoles.includes('viewer')) return 'viewer'
    }

    // AWS Cognito puts roles in cognito:groups
    const cognitoGroups = profile['cognito:groups']
    if (cognitoGroups) {
      if (cognitoGroups.includes('monti-admins')) return 'admin'
      if (cognitoGroups.includes('monti-supervisors')) return 'supervisor'
      if (cognitoGroups.includes('monti-managers')) return 'manager'
      if (cognitoGroups.includes('monti-viewers')) return 'viewer'
    }

    // Check custom groups attribute
    const customGroups = profile['custom:groups']
    if (customGroups) {
      if (customGroups.includes('monti-admins')) return 'admin'
      if (customGroups.includes('monti-supervisors')) return 'supervisor'
      if (customGroups.includes('monti-managers')) return 'manager'
      if (customGroups.includes('monti-viewers')) return 'viewer'
    }

    // Default to viewer
    return 'viewer'
  }

  // Extract groups from token claims
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private extractGroups(profile: any, accessClaims: any): string[] {
    return accessClaims?.groups || profile.groups || profile['cognito:groups'] || profile['custom:groups'] || []
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
