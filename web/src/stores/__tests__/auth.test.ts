import { describe, it, expect, beforeEach } from 'vitest'
import {
  useAuthStore,
  readCurrentUser,
  readCurrentOrg,
  readOrganizations,
} from '../auth'
import { getAuthManager } from '@/lib/wasm-core'

const mgr = () => getAuthManager() as unknown as { _reset: () => void }

describe('useAuthStore', () => {
  beforeEach(() => {
    mgr()._reset()
    useAuthStore.setState({ _tick: 0, _hasHydrated: false, error: null })
  })

  describe('initial state', () => {
    it('should have null user', () => {
      expect(readCurrentUser()).toBeNull()
    })

    it('should have null currentOrg', () => {
      expect(readCurrentOrg()).toBeNull()
    })

    it('should have empty organizations', () => {
      expect(readOrganizations()).toEqual([])
    })
  })

  describe('setAuth', () => {
    it('should set user', () => {
      const user = { id: 1, email: 'test@example.com', username: 'testuser', name: 'Test User' }
      useAuthStore.getState().setAuth('test-token', user)
      expect(readCurrentUser()).toEqual(user)
    })

    it('should handle user without optional fields', () => {
      const user = { id: 1, email: 'test@example.com', username: 'testuser' }
      useAuthStore.getState().setAuth('test-token', user)
      expect(readCurrentUser()).toEqual(user)
      expect(readCurrentUser()?.name).toBeUndefined()
    })
  })

  describe('setOrganizations', () => {
    it('should set organizations', () => {
      const orgs = [
        { id: 1, name: 'Org 1', slug: 'org-1', role: 'owner' },
        { id: 2, name: 'Org 2', slug: 'org-2', role: 'member' },
      ]
      useAuthStore.getState().setOrganizations(orgs)
      expect(readOrganizations()).toEqual(orgs)
    })

    it('should auto-select first org if none selected', () => {
      const orgs = [
        { id: 1, name: 'Org 1', slug: 'org-1', role: 'owner' },
        { id: 2, name: 'Org 2', slug: 'org-2', role: 'member' },
      ]
      useAuthStore.getState().setOrganizations(orgs)
      expect(readCurrentOrg()).toEqual(orgs[0])
    })

    it('should not change currentOrg if already selected', () => {
      const existingOrg = { id: 3, name: 'Existing', slug: 'existing', role: 'admin' }
      useAuthStore.getState().setCurrentOrg(existingOrg)
      const orgs = [
        { id: 1, name: 'Org 1', slug: 'org-1', role: 'owner' },
        { id: 2, name: 'Org 2', slug: 'org-2', role: 'member' },
      ]
      useAuthStore.getState().setOrganizations(orgs)
      expect(readCurrentOrg()).toEqual(existingOrg)
    })

    it('should handle empty organizations array', () => {
      useAuthStore.getState().setOrganizations([])
      expect(readOrganizations()).toEqual([])
      expect(readCurrentOrg()).toBeNull()
    })
  })

  describe('setCurrentOrg', () => {
    it('should set current organization', () => {
      const org = { id: 1, name: 'Test Org', slug: 'test-org', role: 'owner' }
      useAuthStore.getState().setCurrentOrg(org)
      expect(readCurrentOrg()).toEqual(org)
    })
  })

  describe('logout', () => {
    it('should clear all auth state', () => {
      const user = { id: 1, email: 'test@example.com', username: 'testuser' }
      const org = { id: 1, name: 'Org', slug: 'org', role: 'owner' }
      useAuthStore.getState().setAuth('token', user)
      useAuthStore.getState().setOrganizations([org])
      useAuthStore.getState().logout()
      expect(readCurrentUser()).toBeNull()
      expect(readCurrentOrg()).toBeNull()
      expect(readOrganizations()).toEqual([])
    })
  })

  describe('isAuthenticated', () => {
    it('should return false when no user', () => {
      expect(useAuthStore.getState().isAuthenticated()).toBe(false)
    })

    it('should return true when user exists', () => {
      const user = { id: 1, email: 'test@example.com', username: 'testuser' }
      useAuthStore.getState().setAuth('token', user)
      expect(useAuthStore.getState().isAuthenticated()).toBe(true)
    })

    it('should return false after logout', () => {
      const user = { id: 1, email: 'test@example.com', username: 'testuser' }
      useAuthStore.getState().setAuth('token', user)
      useAuthStore.getState().logout()
      expect(useAuthStore.getState().isAuthenticated()).toBe(false)
    })
  })
})
