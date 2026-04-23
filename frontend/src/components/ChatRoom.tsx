import { useEffect, useRef, useState, useCallback } from 'react'
import { useRoomStore } from '../store/room'
import { useWebRTC } from '../hooks/useWebRTC'
import { signalingClient } from '../lib/websocket'
import { useAuthStore } from '../store/auth'
import { useAudioPlayer } from '../hooks/useAudioPlayer'
import { useAudioRecorder } from '../hooks/useAudioRecorder'
import { useConversationState } from '../hooks/useConversationState'
import { useWakeLock } from '../hooks/useWakeLock'

interface AIMessage {
  id: string
  text: string
  isUser: boolean
  audio?: string
  timestamp: Date
}

interface ChatRoomProps {
  onLeave: () => void
}

type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'reconnecting' | 'error'

export function ChatRoom({ onLeave }: ChatRoomProps) {
  const { currentRoom, members, localStream, setCurrentRoom } = useRoomStore()
  const { user: _user } = useAuthStore()

  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>('disconnected')
  const [aiMessages, setAIMessages] = useState<AIMessage[]>([])
  const [aiInput, setAiInput] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [volumeLevel, setVolumeLevel] = useState(0)

  const audioContextRef = useRef<AudioContext | null>(null)
  const analyserRef = useRef<AnalyserNode | null>(null)
  const animationRef = useRef<number | null>(null)

  const {
    state: convState,
    isRecording,
    isProcessing,
    isSpeaking: isAISpeaking,
    startRecording: startConvRecording,
    stopRecording: stopConvRecording,
    interrupt: interruptConv,
    reset: resetConv,
  } = useConversationState()

  const {
    enqueue: enqueueAudio,
    clearQueue: clearAudioQueue,
    stop: stopAudio,
    initAudioContext,
  } = useAudioPlayer({
    onQueueEmpty: () => {
      if (convState === 'speaking') {
        stopConvRecording()
      }
    },
    onError: (err) => {
      console.error('Audio player error:', err)
      setError('音频播放出错')
    },
  })

  const {
    start: startRecorder,
    stop: stopRecorder,
    close: closeRecorder,
    convertToBase64,
  } = useAudioRecorder({
    onData: (data) => {
      const avg = data.reduce((a, b) => a + Math.abs(b), 0) / data.length
      setVolumeLevel(Math.min(avg * 10, 1))
    },
    onError: (err) => {
      console.error('Recorder error:', err)
      setError('录音出错')
    },
  })

  const { getLocalStream, joinRoom, leaveRoom } = useWebRTC({
    onError: (err) => {
      console.error('WebRTC error:', err)
      setError('连接失败')
    },
    onReconnecting: (attempt) => {
      console.log('WebRTC reconnecting, attempt:', attempt)
      setError(`网络不稳定，正在重连... (${attempt})`)
    },
    onReconnected: () => {
      console.log('WebRTC reconnected')
      setError(null)
    },
  })

  const { request: requestWakeLock, isSupported: wakeLockSupported, isActive: wakeLockActive } = useWakeLock()

  useEffect(() => {
    const unsub = signalingClient.onStateChange((state) => {
      setConnectionStatus(state)
    })
    return () => { unsub() }
  }, [])

  useEffect(() => {
    const init = async () => {
      try {
        await getLocalStream()
        if (currentRoom) {
          joinRoom()
        }
      } catch (err) {
        console.error('Failed to initialize:', err)
        setError('无法访问麦克风')
      }
    }

    init()

    signalingClient.on('ai_voice_response', handleAIVoiceResponse)
    signalingClient.on('ai_text_response', handleAITextResponse)
    signalingClient.on('stop_audio', handleStopAudio)
    signalingClient.on('thinking', handleThinking)
    signalingClient.on('ai_text_delta', handleTextDelta)

    return () => {
      leaveRoom()
      signalingClient.disconnect()
      closeRecorder()
      signalingClient.off('ai_voice_response', handleAIVoiceResponse)
      signalingClient.off('ai_text_response', handleAITextResponse)
      signalingClient.off('stop_audio', handleStopAudio)
      signalingClient.off('thinking', handleThinking)
      signalingClient.off('ai_text_delta', handleTextDelta)
    }
  }, [])

  const handleAIVoiceResponse = useCallback((msg: any) => {
    console.log('Received ai_voice_response:', msg)
    const payload = msg.payload || {}
    const audio = payload.audio || ''
    const text = payload.text || ''
    const isFinal = payload.is_final ?? true

    if (!isFinal && audio) {
      enqueueAudio(audio)
    }

    if (isFinal) {
      if (audio) {
        enqueueAudio(audio)
      }
      if (text) {
        setAIMessages(prev => [...prev, {
          id: Date.now().toString(),
          text,
          isUser: false,
          audio,
          timestamp: new Date(),
        }])
      }
    } else if (text) {
      setAIMessages(prev => {
        const last = prev[prev.length - 1]
        if (last && !last.isUser) {
          return [...prev.slice(0, -1), { ...last, text: last.text + text }]
        }
        return [...prev, { id: Date.now().toString(), text, isUser: false, timestamp: new Date() }]
      })
    }
  }, [enqueueAudio])

  const handleAITextResponse = useCallback((msg: any) => {
    console.log('Received ai_text_response:', msg)
    const payload = msg.payload || {}
    setAIMessages(prev => [...prev, {
      id: Date.now().toString(),
      text: payload.text || '',
      isUser: false,
      timestamp: new Date(),
    }])
  }, [])

  const handleStopAudio = useCallback((msg: any) => {
    console.log('Received stop_audio from:', msg)
    clearAudioQueue()
    stopAudio()
  }, [clearAudioQueue, stopAudio])

  const handleThinking = useCallback((msg: any) => {
    console.log('Received thinking:', msg)
    const payload = msg.payload || {}
    if (payload.status === 'recognizing') {
      resetConv()
    } else if (payload.status === 'generating') {
      // AI is generating response
    } else if (payload.status === 'done') {
      // Response complete
    } else if (payload.status === 'no_speech') {
      setError('未检测到语音')
    }
  }, [resetConv])

  const handleTextDelta = useCallback((_msg: any) => {
    // Real-time text display can be implemented here
  }, [])

  useEffect(() => {
    if (localStream) {
      const ctx = new AudioContext()
      const source = ctx.createMediaStreamSource(localStream)
      const analyser = ctx.createAnalyser()
      analyser.fftSize = 256

      source.connect(analyser)
      ctx.resume()

      audioContextRef.current = ctx
      analyserRef.current = analyser

      const checkVolume = () => {
        if (!analyserRef.current) return

        const dataArray = new Uint8Array(analyserRef.current.frequencyBinCount)
        analyserRef.current.getByteFrequencyData(dataArray)

        const avg = dataArray.reduce((a, b) => a + b, 0) / dataArray.length
        setVolumeLevel(avg / 255)

        animationRef.current = requestAnimationFrame(checkVolume)
      }

      checkVolume()

      return () => {
        if (animationRef.current) {
          cancelAnimationFrame(animationRef.current)
        }
        ctx.close()
      }
    }
  }, [localStream])

  const toggleMute = () => {
    if (localStream) {
      localStream.getAudioTracks().forEach((track) => {
        track.enabled = !track.enabled
      })
    }
  }

  const handleStartRecording = async () => {
    try {
      if (wakeLockSupported && !wakeLockActive) {
        await requestWakeLock()
      }
      await initAudioContext()
      await startRecorder()
      startConvRecording()
    } catch (err) {
      console.error('Failed to start recording:', err)
      setError('无法开始录音')
    }
  }

  const handleStopRecording = async () => {
    const pcmData = await stopRecorder()
    stopConvRecording()

    if (pcmData && pcmData.length > 0) {
      const base64 = convertToBase64(pcmData)
      setAIMessages(prev => [...prev, {
        id: Date.now().toString(),
        text: '[Voice Message]',
        isUser: true,
        timestamp: new Date(),
      }])
      signalingClient.sendAIVoiceChat(base64, 16000)
    }
  }

  const handleAITextSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!aiInput.trim() || !signalingClient.isConnected()) {
      return
    }

    const text = aiInput.trim()
    setAIMessages(prev => [...prev, {
      id: Date.now().toString(),
      text,
      isUser: true,
      timestamp: new Date(),
    }])
    setAiInput('')
    signalingClient.sendAITextChat(text)
  }

  const handleInterrupt = () => {
    interruptConv()
    clearAudioQueue()
    stopAudio()
    signalingClient.sendInterrupt()
  }

  const handleLeave = async () => {
    leaveRoom()
    signalingClient.disconnect()
    closeRecorder()
    setCurrentRoom(null)
    onLeave()
  }

  const getStatusText = () => {
    if (connectionStatus === 'reconnecting') return '网络不稳定，正在重连...'
    if (connectionStatus === 'error') return '连接错误'
    if (convState === 'recording') return '正在说话...'
    if (convState === 'processing') return 'AI 思考中...'
    if (convState === 'speaking') return 'AI 说话中...'
    if (convState === 'interrupting') return '正在打断...'
    return ''
  }

  const getStatusColor = () => {
    if (connectionStatus === 'reconnecting' || connectionStatus === 'error') return 'text-red-500'
    if (convState === 'recording') return 'text-green-500'
    if (convState === 'processing' || convState === 'speaking') return 'text-blue-500'
    return 'text-gray-400'
  }

  return (
    <div className="min-h-screen bg-gray-900 flex flex-col">
      <header className="bg-gray-800 border-b border-gray-700 px-4 py-3">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-bold text-white">{currentRoom?.name}</h1>
            <p className={`text-sm ${getStatusColor()}`}>
              {getStatusText() || `${members.length + 1} participant(s)`}
            </p>
          </div>
          <button
            onClick={handleLeave}
            className="bg-red-600 hover:bg-red-700 text-white py-2 px-4 rounded-lg font-medium transition-colors"
          >
            Leave Room
          </button>
        </div>
      </header>

      {error && (
        <div className="bg-red-900/50 border border-red-700 text-red-200 px-4 py-2 mx-4 mt-4 rounded-lg">
          {error}
          <button onClick={() => setError(null)} className="float-right text-red-400 hover:text-red-300">
            Dismiss
          </button>
        </div>
      )}

      <main className="flex-1 p-4">
        <div className="max-w-4xl mx-auto">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
            <div className="bg-gray-800 rounded-lg p-6 border border-gray-700">
              <div className="flex items-center gap-4">
                <div className={`w-12 h-12 rounded-full flex items-center justify-center ${
                  volumeLevel > 0.1 ? 'bg-green-500 animate-pulse' : 'bg-gray-600'
                }`}>
                  <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z" />
                  </svg>
                </div>
                <div>
                  <p className="text-white font-medium">You</p>
                  <p className="text-gray-400 text-sm flex items-center gap-2">
                    <span className={`w-2 h-2 rounded-full ${
                      connectionStatus === 'connected' ? 'bg-green-500' :
                      connectionStatus === 'reconnecting' ? 'bg-yellow-500 animate-pulse' :
                      connectionStatus === 'error' ? 'bg-red-500' : 'bg-gray-500'
                    }`} />
                    {connectionStatus === 'connected' ? 'Connected' : connectionStatus}
                  </p>
                </div>
                <button
                  onClick={toggleMute}
                  className="ml-auto p-3 rounded-full bg-gray-700 hover:bg-gray-600 transition-colors"
                >
                  <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2" />
                  </svg>
                </button>
              </div>

              <div className="mt-4">
                <div className="flex items-center justify-between text-sm text-gray-400 mb-1">
                  <span>Volume</span>
                  <span>{Math.round(volumeLevel * 100)}%</span>
                </div>
                <div className="h-2 bg-gray-700 rounded-full overflow-hidden">
                  <div
                    className={`h-full transition-all duration-100 ${volumeLevel > 0.1 ? 'bg-green-500' : 'bg-blue-500'}`}
                    style={{ width: `${volumeLevel * 100}%` }}
                  />
                </div>
              </div>
            </div>
          </div>

          <div className="bg-gray-800 rounded-lg p-6 border border-gray-700">
            <h3 className="text-lg font-semibold text-white mb-4">AI Assistant</h3>

            <div className="space-y-3 max-h-60 overflow-y-auto mb-4">
              {aiMessages.length === 0 && (
                <p className="text-gray-400 text-sm">Hold the microphone button to speak or type a message</p>
              )}
              {aiMessages.map((msg) => (
                <div key={msg.id} className={`flex ${msg.isUser ? 'justify-end' : 'justify-start'}`}>
                  <div className={`max-w-[80%] rounded-lg px-4 py-2 ${
                    msg.isUser ? 'bg-blue-600 text-white' : 'bg-gray-700 text-white'
                  }`}>
                    <p className="text-sm">{msg.text}</p>
                    {msg.isUser && msg.audio && (
                      <p className="text-xs opacity-70 mt-1">Voice message</p>
                    )}
                  </div>
                </div>
              ))}
              {isProcessing && (
                <div className="flex justify-start">
                  <div className="bg-gray-700 rounded-lg px-4 py-2">
                    <div className="flex gap-1">
                      <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce"></div>
                      <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{animationDelay: '0.2s'}}></div>
                      <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{animationDelay: '0.4s'}}></div>
                    </div>
                  </div>
                </div>
              )}
            </div>

            <form onSubmit={handleAITextSubmit} className="flex gap-2">
              <input
                type="text"
                value={aiInput}
                onChange={(e) => setAiInput(e.target.value)}
                placeholder="Type a message..."
                className="flex-1 bg-gray-700 text-white rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                disabled={isProcessing}
              />
              <button
                type="submit"
                disabled={!aiInput.trim() || isProcessing}
                className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium transition-colors disabled:opacity-50"
              >
                Send
              </button>
            </form>

            <div className="mt-4 flex justify-center items-center gap-4">
              <button
                onMouseDown={handleStartRecording}
                onMouseUp={handleStopRecording}
                onTouchStart={handleStartRecording}
                onTouchEnd={handleStopRecording}
                disabled={isProcessing || connectionStatus !== 'connected'}
                className={`p-4 rounded-full transition-colors ${
                  isRecording
                    ? 'bg-red-500 hover:bg-red-600 animate-pulse'
                    : 'bg-blue-600 hover:bg-blue-700'
                } disabled:opacity-50`}
              >
                <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z" />
                </svg>
              </button>

              {(isProcessing || isAISpeaking) && (
                <button
                  onClick={handleInterrupt}
                  className="p-4 rounded-full bg-red-600 hover:bg-red-700 transition-colors"
                  title="Interrupt"
                >
                  <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" />
                  </svg>
                </button>
              )}

              <p className="text-gray-400 text-sm">
                {isRecording ? 'Recording...' : isProcessing ? 'Processing...' : 'Hold to speak'}
              </p>
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}
