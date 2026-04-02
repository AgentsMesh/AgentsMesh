import { describe, it, expect } from 'vitest'
import { buildPodfileLayer } from '../podfile-layer'

describe('buildPodfileLayer', () => {
  it('generates MODE for non-pty interaction mode', () => {
    const result = buildPodfileLayer({ configValues: {}, interactionMode: 'acp' })
    expect(result).toContain('MODE acp')
  })

  it('omits MODE for pty (default)', () => {
    const result = buildPodfileLayer({ configValues: {}, interactionMode: 'pty' })
    expect(result).not.toContain('MODE')
  })

  it('generates PROMPT declaration', () => {
    const result = buildPodfileLayer({ configValues: {}, prompt: 'fix bug' })
    expect(result).toContain('PROMPT "fix bug"')
  })

  it('escapes PROMPT special characters', () => {
    const result = buildPodfileLayer({
      configValues: {},
      prompt: 'say "hello" and use \\ backslash',
    })
    expect(result).toContain('PROMPT "say \\"hello\\" and use \\\\ backslash"')
  })

  it('generates REPO slug', () => {
    const result = buildPodfileLayer({
      configValues: {},
      repositorySlug: 'dev-org/demo-api',
    })
    expect(result).toContain('REPO "dev-org/demo-api"')
  })

  it('generates BRANCH', () => {
    const result = buildPodfileLayer({ configValues: {}, branchName: 'develop' })
    expect(result).toContain('BRANCH "develop"')
  })

  it('generates CONFIG declarations', () => {
    const result = buildPodfileLayer({ configValues: { model: 'opus' } })
    expect(result).toContain('CONFIG model = "opus"')
  })

  it('generates CREDENTIAL', () => {
    const result = buildPodfileLayer({
      configValues: {},
      credentialProfileName: 'my-profile',
    })
    expect(result).toContain('CREDENTIAL "my-profile"')
  })

  it('returns empty string when all params are empty', () => {
    const result = buildPodfileLayer({ configValues: {} })
    expect(result).toBe('')
  })

  it('generates full output with all fields', () => {
    const result = buildPodfileLayer({
      configValues: { model: 'opus', permission_mode: 'plan' },
      interactionMode: 'acp',
      credentialProfileName: 'my-profile',
      prompt: 'fix the bug',
      repositorySlug: 'dev-org/demo-api',
      branchName: 'develop',
    })
    expect(result).toContain('MODE acp')
    expect(result).toContain('CREDENTIAL "my-profile"')
    expect(result).toContain('PROMPT "fix the bug"')
    expect(result).toContain('CONFIG model = "opus"')
    expect(result).toContain('CONFIG permission_mode = "plan"')
    expect(result).toContain('REPO "dev-org/demo-api"')
    expect(result).toContain('BRANCH "develop"')
  })

  it('skips CONFIG entries with empty string values', () => {
    const result = buildPodfileLayer({ configValues: { model: '', other: 'val' } })
    expect(result).not.toContain('CONFIG model')
    expect(result).toContain('CONFIG other = "val"')
  })

  it('handles CONFIG with boolean values', () => {
    const result = buildPodfileLayer({ configValues: { mcp_enabled: true } })
    expect(result).toContain('CONFIG mcp_enabled = true')
  })

  it('handles CONFIG with numeric values', () => {
    const result = buildPodfileLayer({ configValues: { timeout: 30 } })
    expect(result).toContain('CONFIG timeout = 30')
  })
})
