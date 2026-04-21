import { describe, it, expect, beforeEach } from 'vitest'
import { useUserStore, readProfile, UserProfile, UserIdentity } from '../user'
import { getUserState } from '@/lib/wasm-core'

describe('useUserStore', () => {
  const mockProfile: UserProfile = {
    id: 1,
    email: 'test@example.com',
    username: 'testuser',
    name: 'Test User',
    avatar_url: 'https://example.com/avatar.png',
    is_active: true,
    last_login_at: '2024-01-01T00:00:00Z',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    identities: [],
    organizations: [],
  }

  const mockIdentity: UserIdentity = {
    id: 1,
    provider: 'github',
    provider_user_id: '12345',
    provider_username: 'testuser',
    created_at: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => {
    getUserState().set_profile('')
    useUserStore.setState({ _tick: 0, isLoading: false, error: null })
  })

  describe('initial state', () => {
    it('should have null profile', () => {
      expect(readProfile()).toBeNull()
    })

    it('should have isLoading false', () => {
      expect(useUserStore.getState().isLoading).toBe(false)
    })

    it('should have null error', () => {
      expect(useUserStore.getState().error).toBeNull()
    })
  })

  describe('setProfile', () => {
    it('should set profile', () => {
      useUserStore.getState().setProfile(mockProfile)
      expect(readProfile()).toEqual(mockProfile)
    })

    it('should set profile to null', () => {
      useUserStore.getState().setProfile(mockProfile)
      useUserStore.getState().setProfile(null)
      expect(readProfile()).toBeNull()
    })
  })

  describe('updateProfile', () => {
    it('should update profile fields', () => {
      useUserStore.getState().setProfile(mockProfile)
      useUserStore.getState().updateProfile({ name: 'Updated Name' })
      const p = readProfile()
      expect(p?.name).toBe('Updated Name')
      expect(p?.email).toBe(mockProfile.email)
    })

    it('should update multiple fields', () => {
      useUserStore.getState().setProfile(mockProfile)
      useUserStore.getState().updateProfile({
        name: 'New Name',
        avatar_url: 'https://new-avatar.com/img.png',
      })
      const p = readProfile()
      expect(p?.name).toBe('New Name')
      expect(p?.avatar_url).toBe('https://new-avatar.com/img.png')
    })

    it('should do nothing if profile is null', () => {
      useUserStore.getState().updateProfile({ name: 'Updated Name' })
      expect(readProfile()).toBeNull()
    })
  })

  describe('addIdentity', () => {
    it('should add identity to profile', () => {
      useUserStore.getState().setProfile({ ...mockProfile, identities: [] })
      useUserStore.getState().addIdentity(mockIdentity)
      const p = readProfile()
      expect(p?.identities).toHaveLength(1)
      expect(p?.identities[0]).toEqual(mockIdentity)
    })

    it('should append to existing identities', () => {
      const existingIdentity: UserIdentity = {
        id: 2,
        provider: 'google',
        provider_user_id: '67890',
        created_at: '2024-01-01T00:00:00Z',
      }
      useUserStore.getState().setProfile({ ...mockProfile, identities: [existingIdentity] })
      useUserStore.getState().addIdentity(mockIdentity)
      expect(readProfile()?.identities).toHaveLength(2)
    })

    it('should do nothing if profile is null', () => {
      useUserStore.getState().addIdentity(mockIdentity)
      expect(readProfile()).toBeNull()
    })
  })

  describe('removeIdentity', () => {
    it('should remove identity by provider', () => {
      useUserStore.getState().setProfile({ ...mockProfile, identities: [mockIdentity] })
      useUserStore.getState().removeIdentity('github')
      expect(readProfile()?.identities).toHaveLength(0)
    })

    it('should only remove matching provider', () => {
      const googleIdentity: UserIdentity = {
        id: 2,
        provider: 'google',
        provider_user_id: '67890',
        created_at: '2024-01-01T00:00:00Z',
      }
      useUserStore.getState().setProfile({ ...mockProfile, identities: [mockIdentity, googleIdentity] })
      useUserStore.getState().removeIdentity('github')
      const p = readProfile()
      expect(p?.identities).toHaveLength(1)
      expect(p?.identities[0].provider).toBe('google')
    })

    it('should do nothing if profile is null', () => {
      useUserStore.getState().removeIdentity('github')
      expect(readProfile()).toBeNull()
    })
  })

  describe('setLoading', () => {
    it('should set loading to true', () => {
      useUserStore.getState().setLoading(true)
      expect(useUserStore.getState().isLoading).toBe(true)
    })

    it('should set loading to false', () => {
      useUserStore.setState({ isLoading: true })
      useUserStore.getState().setLoading(false)
      expect(useUserStore.getState().isLoading).toBe(false)
    })
  })

  describe('setError', () => {
    it('should set error message', () => {
      useUserStore.getState().setError('Test error')
      expect(useUserStore.getState().error).toBe('Test error')
    })

    it('should clear error', () => {
      useUserStore.setState({ error: 'Previous error' })
      useUserStore.getState().setError(null)
      expect(useUserStore.getState().error).toBeNull()
    })
  })

  describe('reset', () => {
    it('should reset to initial state', () => {
      useUserStore.getState().setProfile(mockProfile)
      useUserStore.setState({ isLoading: true, error: 'Some error' })
      useUserStore.getState().reset()
      expect(readProfile()).toBeNull()
      expect(useUserStore.getState().isLoading).toBe(false)
      expect(useUserStore.getState().error).toBeNull()
    })
  })
})
