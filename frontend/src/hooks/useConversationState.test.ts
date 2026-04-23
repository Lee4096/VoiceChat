import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useConversationState } from '../hooks/useConversationState'

describe('useConversationState', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('initial state', () => {
    it('should start with idle state', () => {
      const { result } = renderHook(() => useConversationState())
      expect(result.current.state).toBe('idle')
      expect(result.current.isIdle).toBe(true)
      expect(result.current.isActive).toBe(false)
    })

    it('should have all state flags as false except idle', () => {
      const { result } = renderHook(() => useConversationState())
      expect(result.current.isRecording).toBe(false)
      expect(result.current.isProcessing).toBe(false)
      expect(result.current.isSpeaking).toBe(false)
      expect(result.current.isInterrupting).toBe(false)
      expect(result.current.isError).toBe(false)
    })
  })

  describe('state transitions', () => {
    it('should transition from idle to recording', () => {
      const { result } = renderHook(() => useConversationState())

      act(() => {
        result.current.startRecording()
      })

      expect(result.current.state).toBe('recording')
      expect(result.current.isRecording).toBe(true)
      expect(result.current.isActive).toBe(true)
    })

    it('should transition from recording to processing', () => {
      const { result } = renderHook(() => useConversationState())

      act(() => {
        result.current.startRecording()
      })

      act(() => {
        result.current.stopRecording()
      })

      expect(result.current.state).toBe('processing')
      expect(result.current.isProcessing).toBe(true)
    })

    it('should transition from processing to speaking', () => {
      const { result } = renderHook(() => useConversationState())

      act(() => {
        result.current.startRecording()
      })

      act(() => {
        result.current.stopRecording()
      })

      act(() => {
        result.current.startSpeaking()
      })

      expect(result.current.state).toBe('speaking')
      expect(result.current.isSpeaking).toBe(true)
    })

    it('should transition from speaking to idle', () => {
      const { result } = renderHook(() => useConversationState())

      act(() => {
        result.current.startRecording()
      })
      act(() => {
        result.current.stopRecording()
      })
      act(() => {
        result.current.startSpeaking()
      })
      act(() => {
        result.current.stopSpeaking()
      })

      expect(result.current.state).toBe('idle')
      expect(result.current.isIdle).toBe(true)
    })

    it('should transition to interrupting from recording', () => {
      const { result } = renderHook(() => useConversationState())

      act(() => {
        result.current.startRecording()
      })

      act(() => {
        result.current.interrupt()
      })

      expect(result.current.state).toBe('interrupting')
    })

    it('should transition to interrupting from processing', () => {
      const { result } = renderHook(() => useConversationState())

      act(() => {
        result.current.startRecording()
      })
      act(() => {
        result.current.stopRecording()
      })

      act(() => {
        result.current.interrupt()
      })

      expect(result.current.state).toBe('interrupting')
    })

    it('should transition to interrupting from speaking', () => {
      const { result } = renderHook(() => useConversationState())

      act(() => {
        result.current.startRecording()
      })
      act(() => {
        result.current.stopRecording()
      })
      act(() => {
        result.current.startSpeaking()
      })

      act(() => {
        result.current.interrupt()
      })

      expect(result.current.state).toBe('interrupting')
    })
  })

  describe('reset', () => {
    it('should reset to idle state', () => {
      const { result } = renderHook(() => useConversationState())

      act(() => {
        result.current.startRecording()
      })
      act(() => {
        result.current.reset()
      })

      expect(result.current.state).toBe('idle')
    })

    it('should clear state history', () => {
      const { result } = renderHook(() => useConversationState())

      act(() => {
        result.current.startRecording()
      })
      act(() => {
        result.current.stopRecording()
      })
      act(() => {
        result.current.reset()
      })

      const history = result.current.getStateHistory()
      expect(history).toEqual(['idle'])
    })
  })

  describe('state history', () => {
    it('should track state history', () => {
      const { result } = renderHook(() => useConversationState())

      act(() => {
        result.current.startRecording()
      })
      act(() => {
        result.current.stopRecording()
      })

      const history = result.current.getStateHistory()
      expect(history).toContain('idle')
      expect(history).toContain('recording')
      expect(history).toContain('processing')
    })
  })

  describe('error state', () => {
    it('should transition to error state', () => {
      const { result } = renderHook(() => useConversationState())

      act(() => {
        result.current.setError()
      })

      expect(result.current.state).toBe('error')
      expect(result.current.isError).toBe(true)
    })

    it('should only allow transition from error to idle', () => {
      const { result } = renderHook(() => useConversationState())

      act(() => {
        result.current.setError()
      })

      expect(result.current.canTransition('idle')).toBe(true)
      expect(result.current.canTransition('recording')).toBe(false)
      expect(result.current.canTransition('processing')).toBe(false)
      expect(result.current.canTransition('speaking')).toBe(false)
    })
  })

  describe('onStateChange callback', () => {
    it('should call onStateChange when state changes', () => {
      const onStateChange = vi.fn()

      const { result } = renderHook(() =>
        useConversationState({ onStateChange })
      )

      act(() => {
        result.current.startRecording()
      })

      expect(onStateChange).toHaveBeenCalledWith('recording', 'idle')
    })

    it('should call onTransition when state changes', () => {
      const onTransition = vi.fn()

      const { result } = renderHook(() =>
        useConversationState({ onTransition })
      )

      act(() => {
        result.current.startRecording()
      })

      expect(onTransition).toHaveBeenCalledWith({
        from: 'idle',
        to: 'recording',
        reason: 'user_action',
      })
    })
  })
})
