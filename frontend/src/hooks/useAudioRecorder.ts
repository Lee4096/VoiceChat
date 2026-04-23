import { useCallback, useRef, useState } from 'react'

export type RecorderState = 'idle' | 'recording' | 'paused'

interface UseAudioRecorderOptions {
  sampleRate?: number
  bufferSize?: number
  onData?: (data: Float32Array) => void
  onError?: (error: Error) => void
}

export function useAudioRecorder(options: UseAudioRecorderOptions = {}) {
  const {
    sampleRate = 16000,
    bufferSize = 4096,
    onData,
    onError,
  } = options

  const audioContextRef = useRef<AudioContext | null>(null)
  const mediaStreamRef = useRef<MediaStream | null>(null)
  const scriptProcessorRef = useRef<ScriptProcessorNode | null>(null)
  const isRecordingRef = useRef(false)
  const [state, setState] = useState<RecorderState>('idle')
  const recordingChunksRef = useRef<Float32Array[]>([])

  const getUserMedia = useCallback(async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
          sampleRate: sampleRate,
        },
        video: false,
      })
      mediaStreamRef.current = stream
      return stream
    } catch (error) {
      onError?.(error as Error)
      return null
    }
  }, [sampleRate, onError])

  const createAudioContext = useCallback(async () => {
    if (!audioContextRef.current) {
      audioContextRef.current = new AudioContext({ sampleRate })
    }

    if (audioContextRef.current.state === 'suspended') {
      await audioContextRef.current.resume()
    }

    return audioContextRef.current
  }, [sampleRate])

  const start = useCallback(async () => {
    if (state === 'recording') {
      return
    }

    const stream = mediaStreamRef.current || await getUserMedia()
    if (!stream) {
      onError?.(new Error('Failed to get microphone access'))
      return false
    }

    try {
      const audioContext = await createAudioContext()
      const source = audioContext.createMediaStreamSource(stream)
      const processor = audioContext.createScriptProcessor(bufferSize, 1, 1)

      recordingChunksRef.current = []
      isRecordingRef.current = true

      processor.onaudioprocess = (e) => {
        if (!isRecordingRef.current) return
        const inputData = e.inputBuffer.getChannelData(0)
        const copy = new Float32Array(inputData)
        recordingChunksRef.current.push(copy)
        onData?.(copy)
      }

      source.connect(processor)
      processor.connect(audioContext.destination)

      scriptProcessorRef.current = processor
      setState('recording')

      return true
    } catch (error) {
      onError?.(error as Error)
      return false
    }
  }, [bufferSize, createAudioContext, getUserMedia, onData, onError])

  const stop = useCallback(async () => {
    if (state !== 'recording') {
      return null
    }

    isRecordingRef.current = false

    if (scriptProcessorRef.current) {
      scriptProcessorRef.current.disconnect()
      scriptProcessorRef.current = null
    }

    setState('idle')

    const chunks = recordingChunksRef.current
    if (chunks.length === 0) {
      return null
    }

    const totalLength = chunks.reduce((acc, chunk) => acc + chunk.length, 0)
    const pcmData = new Float32Array(totalLength)
    let offset = 0
    for (const chunk of chunks) {
      pcmData.set(chunk, offset)
      offset += chunk.length
    }

    recordingChunksRef.current = []

    return pcmData
  }, [state])

  const pause = useCallback(() => {
    if (state === 'recording') {
      setState('paused')
    }
  }, [state])

  const resume = useCallback(() => {
    if (state === 'paused') {
      setState('recording')
    }
  }, [state])

  const suspend = useCallback(() => {
    if (audioContextRef.current && audioContextRef.current.state === 'running') {
      audioContextRef.current.suspend()
    }
  }, [])

  const close = useCallback(() => {
    if (scriptProcessorRef.current) {
      scriptProcessorRef.current.disconnect()
      scriptProcessorRef.current = null
    }

    if (mediaStreamRef.current) {
      mediaStreamRef.current.getTracks().forEach(track => track.stop())
      mediaStreamRef.current = null
    }

    if (audioContextRef.current) {
      audioContextRef.current.close()
      audioContextRef.current = null
    }

    recordingChunksRef.current = []
    setState('idle')
  }, [])

  const convertToBase64 = useCallback((pcmData: Float32Array): string => {
    const int16Data = new Int16Array(pcmData.length)
    for (let i = 0; i < pcmData.length; i++) {
      const s = Math.max(-1, Math.min(1, pcmData[i]))
      int16Data[i] = s < 0 ? s * 0x8000 : s * 0x7FFF
    }

    const bytes = new Uint8Array(int16Data.buffer)
    let binary = ''
    for (let i = 0; i < bytes.length; i++) {
      binary += String.fromCharCode(bytes[i])
    }
    return btoa(binary)
  }, [])

  return {
    state,
    start,
    stop,
    pause,
    resume,
    suspend,
    close,
    convertToBase64,
    getUserMedia,
  }
}
