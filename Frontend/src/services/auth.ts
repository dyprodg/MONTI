import { UserManager, User as OidcUser, WebStorageStateStore } from 'oidc-client-ts'

export interface User {
  email: string
  name: string
  role: string
  groups: string[]
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

    return {
      email: (profile.email as string) || 'unknown@monti.local',
      name: (profile.name as string) || (profile.preferred_username as string) || 'Unknown User',
      role: this.extractRole(profile),
      groups: this.extractGroups(profile),
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
}

export const authService = new AuthService()
