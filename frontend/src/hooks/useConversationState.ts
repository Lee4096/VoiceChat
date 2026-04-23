import { useCallback, useRef, useState } from 'react'

export type ConversationState =
  | 'idle'
  | 'recording'
  | 'processing'
  | 'speaking'
  | 'interrupting'
  | 'error'

export type StateReason =
  | 'user_action'
  | 'server_response'
  | 'network_error'
  | 'timeout'
  | 'manual'

interface StateTransition {
  from: ConversationState
  to: ConversationState
  reason: StateReason
}

interface UseConversationStateOptions {
  onStateChange?: (state: ConversationState, prev: ConversationState) => void
  onTransition?: (transition: StateTransition) => void
}

const VALID_TRANSITIONS: Record<ConversationState, ConversationState[]> = {
  idle: ['recording', 'error'],
  recording: ['processing', 'idle', 'interrupting', 'error'],
  processing: ['speaking', 'idle', 'interrupting', 'error'],
  speaking: ['idle', 'recording', 'interrupting', 'error'],
  interrupting: ['idle', 'recording', 'error'],
  error: ['idle'],
}

export function useConversationState(options: UseConversationStateOptions = {}) {
  const [state, setState] = useState<ConversationState>('idle')
  const [previousState, setPreviousState] = useState<ConversationState>('idle')
  const [lastTransition, setLastTransition] = useState<StateTransition | null>(null)
  const stateHistoryRef = useRef<ConversationState[]>(['idle'])

  const transitionTo = useCallback((newState: ConversationState, reason: StateReason = 'user_action') => {
    setPreviousState(state)
    setState(newState)

    const transition: StateTransition = {
      from: state,
      to: newState,
      reason,
    }

    setLastTransition(transition)
    stateHistoryRef.current.push(newState)

    options.onStateChange?.(newState, state)
    options.onTransition?.(transition)
  }, [state, options])

  const canTransition = useCallback((targetState: ConversationState): boolean => {
    return VALID_TRANSITIONS[state]?.includes(targetState) ?? false
  }, [state])

  const startRecording = useCallback(() => {
    if (state === 'idle' || state === 'speaking' || state === 'error') {
      transitionTo('recording', 'user_action')
    }
  }, [state, transitionTo])

  const stopRecording = useCallback(() => {
    if (state === 'recording') {
      transitionTo('processing', 'user_action')
    }
  }, [state, transitionTo])

  const startProcessing = useCallback(() => {
    if (state === 'recording' || state === 'processing') {
      transitionTo('processing', 'server_response')
    }
  }, [state, transitionTo])

  const startSpeaking = useCallback(() => {
    if (state === 'processing') {
      transitionTo('speaking', 'server_response')
    }
  }, [state, transitionTo])

  const stopSpeaking = useCallback(() => {
    if (state === 'speaking' || state === 'processing') {
      transitionTo('idle', 'server_response')
    }
  }, [state, transitionTo])

  const interrupt = useCallback(() => {
    if (state === 'recording' || state === 'processing' || state === 'speaking') {
      transitionTo('interrupting', 'user_action')
      setTimeout(() => {
        transitionTo('idle', 'user_action')
      }, 100)
    }
  }, [state, transitionTo])

  const setError = useCallback(() => {
    transitionTo('error', 'network_error')
  }, [transitionTo])

  const reset = useCallback(() => {
    transitionTo('idle', 'manual')
    stateHistoryRef.current = ['idle']
  }, [transitionTo])

  const getStateHistory = useCallback(() => {
    return [...stateHistoryRef.current]
  }, [])

  const isIdle = state === 'idle'
  const isRecording = state === 'recording'
  const isProcessing = state === 'processing'
  const isSpeaking = state === 'speaking'
  const isInterrupting = state === 'interrupting'
  const isError = state === 'error'
  const isActive = state === 'recording' || state === 'processing' || state === 'speaking'

  return {
    state,
    previousState,
    lastTransition,
    canTransition,
    startRecording,
    stopRecording,
    startProcessing,
    startSpeaking,
    stopSpeaking,
    interrupt,
    setError,
    reset,
    getStateHistory,
    isIdle,
    isRecording,
    isProcessing,
    isSpeaking,
    isInterrupting,
    isError,
    isActive,
  }
}
