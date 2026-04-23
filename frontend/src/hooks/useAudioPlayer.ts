import { useCallback, useRef, useState } from 'react'

export type AudioPlayerState = 'idle' | 'buffering' | 'playing' | 'paused'

interface QueuedAudio {
  id: string
  audioBuffer: AudioBuffer
}

interface UseAudioPlayerOptions {
  onStateChange?: (state: AudioPlayerState) => void
  onQueueEmpty?: () => void
  onError?: (error: Error) => void
}

export function useAudioPlayer(options: UseAudioPlayerOptions = {}) {
  const audioContextRef = useRef<AudioContext | null>(null)
  const currentSourceRef = useRef<AudioBufferSourceNode | null>(null)
  const queueRef = useRef<QueuedAudio[]>([])
  const isPlayingRef = useRef(false)
  const [state, setState] = useState<AudioPlayerState>('idle')
  const [queueLength, setQueueLength] = useState(0)

  const updateState = useCallback((newState: AudioPlayerState) => {
    setState(newState)
    options.onStateChange?.(newState)
  }, [options.onStateChange])

  const getAudioContext = useCallback(() => {
    if (!audioContextRef.current) {
      audioContextRef.current = new AudioContext()
    }
    return audioContextRef.current
  }, [])

  const initAudioContext = useCallback(async () => {
    const ctx = getAudioContext()
    if (ctx.state === 'suspended') {
      await ctx.resume()
    }
    return ctx
  }, [getAudioContext])

  const playNextInQueue = useCallback(async () => {
    if (queueRef.current.length === 0) {
      isPlayingRef.current = false
      updateState('idle')
      options.onQueueEmpty?.()
      return
    }

    const next = queueRef.current.shift()
    setQueueLength(queueRef.current.length)

    if (!next) {
      return
    }

    try {
      await initAudioContext()
      const ctx = audioContextRef.current!

      if (currentSourceRef.current) {
        currentSourceRef.current.stop()
        currentSourceRef.current.disconnect()
      }

      const source = ctx.createBufferSource()
      source.buffer = next.audioBuffer
      source.connect(ctx.destination)

      source.onended = () => {
        playNextInQueue()
      }

      currentSourceRef.current = source
      isPlayingRef.current = true
      updateState('playing')

      source.start(0)
    } catch (error) {
      options.onError?.(error as Error)
      playNextInQueue()
    }
  }, [initAudioContext, updateState, options])

  const decodeAudioData = useCallback(async (audioData: ArrayBuffer): Promise<AudioBuffer> => {
    const ctx = getAudioContext()
    return ctx.decodeAudioData(audioData)
  }, [getAudioContext])

  const enqueue = useCallback(async (audioData: ArrayBuffer | string) => {
    let arrayBuffer: ArrayBuffer

    if (typeof audioData === 'string') {
      const binary = atob(audioData)
      const bytes = new Uint8Array(binary.length)
      for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i)
      }
      arrayBuffer = bytes.buffer
    } else {
      arrayBuffer = audioData
    }

    try {
      const audioBuffer = await decodeAudioData(arrayBuffer)
      const id = `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`

      queueRef.current.push({ id, audioBuffer })
      setQueueLength(queueRef.current.length)

      if (queueRef.current.length === 1 && !isPlayingRef.current) {
        updateState('buffering')
        playNextInQueue()
      }
    } catch (error) {
      options.onError?.(error as Error)
    }
  }, [decodeAudioData, playNextInQueue, updateState, options])

  const clearQueue = useCallback(() => {
    if (currentSourceRef.current) {
      currentSourceRef.current.stop()
      currentSourceRef.current.disconnect()
      currentSourceRef.current = null
    }

    queueRef.current = []
    setQueueLength(0)
    isPlayingRef.current = false
    updateState('idle')
  }, [updateState])

  const stop = useCallback(() => {
    if (currentSourceRef.current) {
      currentSourceRef.current.stop()
      currentSourceRef.current.disconnect()
      currentSourceRef.current = null
    }
    clearQueue()
  }, [clearQueue])

  const pause = useCallback(() => {
    if (audioContextRef.current && audioContextRef.current.state === 'running') {
      audioContextRef.current.suspend()
      updateState('paused')
    }
  }, [updateState])

  const resume = useCallback(async () => {
    if (audioContextRef.current && audioContextRef.current.state === 'suspended') {
      await audioContextRef.current.resume()
      updateState('playing')
    }
  }, [updateState])

  const suspend = useCallback(() => {
    if (audioContextRef.current && audioContextRef.current.state === 'running') {
      audioContextRef.current.suspend()
    }
  }, [])

  const getQueueLength = useCallback(() => {
    return queueRef.current.length
  }, [])

  return {
    state,
    queueLength,
    enqueue,
    clearQueue,
    stop,
    pause,
    resume,
    suspend,
    getQueueLength,
    initAudioContext,
    isPlaying: isPlayingRef.current,
  }
}
