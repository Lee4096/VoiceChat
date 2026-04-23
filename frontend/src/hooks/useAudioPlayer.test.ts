import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useAudioPlayer } from '../hooks/useAudioPlayer'

describe('useAudioPlayer', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.stubGlobal('AudioContext', vi.fn().mockImplementation(() => ({
      state: 'running',
      createBufferSource: () => ({
        buffer: null,
        connect: vi.fn(),
        start: vi.fn(),
        stop: vi.fn(),
        onended: null,
      }),
      destination: {},
      decodeAudioData: vi.fn().mockResolvedValue({
        duration: 1,
        sampleRate: 16000,
        length: 16000,
        numberOfChannels: 1,
        getChannelData: () => new Float32Array(16000),
      }),
      resume: vi.fn().mockResolvedValue(undefined),
      suspend: vi.fn(),
      close: vi.fn(),
    })))
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('initial state', () => {
    it('should start with idle state', () => {
      const { result } = renderHook(() => useAudioPlayer())
      expect(result.current.state).toBe('idle')
      expect(result.current.isPlaying).toBe(false)
      expect(result.current.queueLength).toBe(0)
    })

    it('should have queueLength of 0', () => {
      const { result } = renderHook(() => useAudioPlayer())
      expect(result.current.queueLength).toBe(0)
    })
  })

  describe('clearQueue', () => {
    it('should have queueLength of 0 when cleared', () => {
      const { result } = renderHook(() => useAudioPlayer())

      result.current.clearQueue()

      expect(result.current.queueLength).toBe(0)
    })

    it('should reset state to idle after clearQueue', () => {
      const { result } = renderHook(() => useAudioPlayer())

      result.current.clearQueue()

      expect(result.current.state).toBe('idle')
    })
  })

  describe('getQueueLength', () => {
    it('should return 0 for empty queue', () => {
      const { result } = renderHook(() => useAudioPlayer())
      expect(result.current.getQueueLength()).toBe(0)
    })
  })

  describe('stop', () => {
    it('should clear queue on stop', () => {
      const { result } = renderHook(() => useAudioPlayer())

      result.current.stop()

      expect(result.current.queueLength).toBe(0)
    })
  })
})
