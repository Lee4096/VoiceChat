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
  const workletRef = useRef<AudioWorkletNode | null>(null)
  const isRecordingRef = useRef(false)
  const [state, setState] = useState<RecorderState>('idle')
  const recordingChunksRef = useRef<Float32Array[]>([])
  const actualSampleRateRef = useRef<number>(16000)

  const getUserMedia = useCallback(async () => {
    console.log('[Recorder] getUserMedia called')
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
        },
        video: false,
      })
      console.log('[Recorder] getUserMedia success, stream:', stream.active)
      mediaStreamRef.current = stream
      return stream
    } catch (error) {
      console.error('[Recorder] getUserMedia error:', error)
      onError?.(error as Error)
      return null
    }
  }, [onError])

  const createAudioContext = useCallback(async () => {
    if (!audioContextRef.current) {
      // 不指定 sampleRate，使用系统默认采样率
      audioContextRef.current = new AudioContext()
    }
    console.log('[Recorder] AudioContext sampleRate:', audioContextRef.current.sampleRate)
    console.log('[Recorder] AudioContext state:', audioContextRef.current.state)

    if (audioContextRef.current.state === 'suspended') {
      console.log('[Recorder] AudioContext suspended, resuming...')
      await audioContextRef.current.resume()
      console.log('[Recorder] AudioContext resumed, new state:', audioContextRef.current.state)
    }

    // 加载 AudioWorklet
    if (!audioContextRef.current.audioWorklet) {
      console.log('[Recorder] AudioWorklet not supported')
      throw new Error('AudioWorklet not supported')
    }

    try {
      await audioContextRef.current.audioWorklet.addModule(
        new URL('./recording-processor.js', import.meta.url).href
      )
      console.log('[Recorder] AudioWorklet loaded')
    } catch (err) {
      console.error('[Recorder] Failed to load AudioWorklet:', err)
      throw err
    }

    return audioContextRef.current
  }, [sampleRate])

  const start = useCallback(async () => {
    console.log('[Recorder] start called, current state:', state)
    console.log('[Recorder] mediaStreamRef.current:', mediaStreamRef.current ? `exists (active: ${mediaStreamRef.current.active})` : 'null')
    if (state === 'recording') {
      console.log('[Recorder] Already recording, returning')
      return
    }

    // 检查现有 stream 是否有效
    let stream = mediaStreamRef.current
    if (!stream || !stream.active) {
      console.log('[Recorder] Stream is null or inactive, calling getUserMedia')
      stream = await getUserMedia()
    } else {
      console.log('[Recorder] Reusing existing active stream')
    }

    if (!stream) {
      console.error('[Recorder] No stream available')
      onError?.(new Error('Failed to get microphone access'))
      return false
    }

    try {
      const audioContext = await createAudioContext()
      console.log('[Recorder] AudioContext created')
      const source = audioContext.createMediaStreamSource(stream)
      console.log('[Recorder] source created')

      // 使用 AudioWorkletNode
      const worklet = new AudioWorkletNode(audioContext, 'recording-processor', {
        processorOptions: { bufferSize }
      })
      console.log('[Recorder] AudioWorkletNode created')

      recordingChunksRef.current = []
      isRecordingRef.current = true
      console.log('[Recorder] Recording started, waiting for audio data...')

      // 处理来自 worklet 的消息
      worklet.port.onmessage = (event) => {
        console.log('[Recorder] AudioWorklet message received')
        if (!isRecordingRef.current) return
        const buffer = event.data.buffer
        if (buffer) {
          // 检查音频数据是否有内容
          let sum = 0
          for (let i = 0; i < buffer.length; i++) {
            sum += Math.abs(buffer[i])
          }
          const avg = sum / buffer.length
          console.log('[Recorder] Buffer length:', buffer.length, ', avg amplitude:', avg.toFixed(6))
          recordingChunksRef.current.push(buffer)
          onData?.(buffer)
        }
      }

      source.connect(worklet)
      // 不连接 destination，避免听到自己的声音

      workletRef.current = worklet
      setState('recording')
      console.log('[Recorder] State set to recording')

      return true
    } catch (error) {
      console.error('[Recorder] Error:', error)
      onError?.(error as Error)
      return false
    }
  }, [bufferSize, createAudioContext, getUserMedia, onData, onError, state])

  const resample = useCallback((input: Float32Array, inputRate: number, outputRate: number): Float32Array => {
    if (inputRate === outputRate) {
      return input
    }
    const ratio = inputRate / outputRate
    const outputLength = Math.round(input.length / ratio)
    const output = new Float32Array(outputLength)

    for (let i = 0; i < outputLength; i++) {
      const srcIdx = i * ratio
      const srcIdxFloor = Math.floor(srcIdx)
      const srcIdxCeil = Math.ceil(srcIdx)

      if (srcIdxCeil >= input.length) {
        output[i] = input[input.length - 1]
      } else if (srcIdxFloor === srcIdxCeil || srcIdxFloor >= input.length) {
        output[i] = input[srcIdxFloor]
      } else {
        const fraction = srcIdx - srcIdxFloor
        output[i] = input[srcIdxFloor] * (1 - fraction) + input[srcIdxCeil] * fraction
      }
    }
    return output
  }, [])

  const stop = useCallback(async () => {
    console.log('[Recorder] stop called, current state:', state)
    if (state !== 'recording') {
      console.log('[Recorder] Not recording, returning null')
      return null
    }

    isRecordingRef.current = false
    console.log('[Recorder] isRecordingRef set to false')

    if (workletRef.current) {
      workletRef.current.port.postMessage('stop')
      workletRef.current.disconnect()
      workletRef.current = null
    }

    setState('idle')

    const chunks = recordingChunksRef.current
    console.log('[Recorder] chunks collected:', chunks.length)
    if (chunks.length === 0) {
      console.log('[Recorder] No chunks, returning null')
      return null
    }

    const totalLength = chunks.reduce((acc, chunk) => acc + chunk.length, 0)
    console.log('[Recorder] Total samples:', totalLength)
    const pcmData = new Float32Array(totalLength)
    let offset = 0
    for (const chunk of chunks) {
      pcmData.set(chunk, offset)
      offset += chunk.length
    }

    recordingChunksRef.current = []

    const targetSampleRate = 16000
    const actualSampleRate = audioContextRef.current?.sampleRate ?? targetSampleRate
    console.log('[Recorder] Actual sample rate:', actualSampleRate, ', Target:', targetSampleRate)

    if (actualSampleRate !== targetSampleRate) {
      console.log('[Recorder] Resampling from', actualSampleRate, 'to', targetSampleRate)
      const resampled = resample(pcmData, actualSampleRate, targetSampleRate)
      actualSampleRateRef.current = targetSampleRate
      return resampled
    }

    actualSampleRateRef.current = actualSampleRate
    return pcmData
  }, [state, resample])

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
    if (workletRef.current) {
      workletRef.current.disconnect()
      workletRef.current = null
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

  const getSampleRate = useCallback(() => {
    return actualSampleRateRef.current
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
    getSampleRate,
  }
}
