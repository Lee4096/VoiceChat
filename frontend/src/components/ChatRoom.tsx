import { useEffect, useRef, useState, useCallback } from 'react'
import { useRoomStore } from '../store/room'
import { useWebRTC } from '../hooks/useWebRTC'
import { signalingClient } from '../lib/websocket'
import { useAuthStore } from '../store/auth'

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

export function ChatRoom({ onLeave }: ChatRoomProps) {
  const { currentRoom, members, localStream, remoteStreams, setCurrentRoom, connectionState } = useRoomStore()
  const { user } = useAuthStore()
  const [isMuted, setIsMuted] = useState(false)
  const [isSpeaking, setIsSpeaking] = useState(false)
  const [aiMessages, setAIMessages] = useState<AIMessage[]>([])
  const [aiInput, setAiInput] = useState('')
  const [isAITyping, setIsAITyping] = useState(false)
  const [isRecording, setIsRecording] = useState(false)
  const isRecordingRef = useRef(false)
  const audioContextRef = useRef<AudioContext | null>(null)
  const analyserRef = useRef<AnalyserNode | null>(null)
  const animationRef = useRef<number | null>(null)
  const mediaRecorderRef = useRef<MediaRecorder | null>(null)
  const audioChunksRef = useRef<Blob[]>([])
  const recordingChunksRef = useRef<Float32Array[]>([])
  const scriptProcessorRef = useRef<ScriptProcessorNode | null>(null)

  const { getLocalStream, joinRoom, leaveRoom } = useWebRTC({
    onError: (error) => {
      console.error('WebRTC error:', error)
    },
  })

  useEffect(() => {
    const init = async () => {
      try {
        await getLocalStream()
        if (currentRoom) {
          joinRoom()
        }
      } catch (error) {
        console.error('Failed to initialize:', error)
      }
    }

    init()

    // Listen for AI responses
    const handleAIVoiceResponse = (msg: any) => {
      console.log('Received ai_voice_response:', msg)
      setIsAITyping(false)
      const payload = msg.payload || {}
      const audio = payload.audio || ''
      const text = payload.text || ''
      setAIMessages(prev => [...prev, {
        id: Date.now().toString(),
        text: text,
        isUser: false,
        audio: audio,
        timestamp: new Date(),
      }])
      if (audio) {
        try {
          const int16Array = base64ToInt16Array(audio)
          const wavBlob = createWavBlob(int16Array, 16000)
          const audioUrl = URL.createObjectURL(wavBlob)
          const audioElement = new Audio(audioUrl)
          audioElement.play().catch(e => console.error('Audio play error:', e))
        } catch (e) {
          console.error('Error playing audio:', e)
        }
      }
    }

    const handleAITextResponse = (msg: any) => {
      console.log('Received ai_text_response:', msg)
      setIsAITyping(false)
      const payload = msg.payload || {}
      setAIMessages(prev => [...prev, {
        id: Date.now().toString(),
        text: payload.text || '',
        isUser: false,
        timestamp: new Date(),
      }])
    }

    const handleStopAudio = (msg: any) => {
      console.log('Received stop_audio from:', msg)
      setIsAITyping(false)
    }

    signalingClient.on('ai_voice_response', handleAIVoiceResponse as any)
    signalingClient.on('ai_text_response', handleAITextResponse as any)
    signalingClient.on('stop_audio', handleStopAudio as any)

    return () => {
      leaveRoom()
      signalingClient.disconnect()
      signalingClient.off('ai_voice_response', handleAIVoiceResponse as any)
      signalingClient.off('ai_text_response', handleAITextResponse as any)
      signalingClient.off('stop_audio', handleStopAudio as any)
    }
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

        const average = dataArray.reduce((a, b) => a + b, 0) / dataArray.length
        setIsSpeaking(average > 30)

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
      setIsMuted(!isMuted)
    }
  }

  const base64ToBlob = (base64: string, mimeType: string): Blob => {
    const byteCharacters = atob(base64)
    const byteNumbers = new Array(byteCharacters.length)
    for (let i = 0; i < byteCharacters.length; i++) {
      byteNumbers[i] = byteCharacters.charCodeAt(i)
    }
    const byteArray = new Uint8Array(byteNumbers)
    return new Blob([byteArray], { type: mimeType })
  }

  const handleAITextSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    console.log('handleAITextSubmit called, input:', aiInput, 'connected:', signalingClient.isConnected())
    if (!aiInput.trim() || !signalingClient.isConnected()) {
      console.log('Cannot send: not connected or empty input')
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
    setIsAITyping(true)
    console.log('Sending AI text chat:', text)
    signalingClient.sendAITextChat(text)
  }

  const startRecording = useCallback(async () => {
    console.log('startRecording called, localStream:', !!localStream, 'connected:', signalingClient.isConnected())
    if (!localStream || !signalingClient.isConnected()) return

    recordingChunksRef.current = []
    isRecordingRef.current = true

    const audioContext = new AudioContext({ sampleRate: 16000 })
    const source = audioContext.createMediaStreamSource(localStream)
    const processor = audioContext.createScriptProcessor(4096, 1, 1)

    processor.onaudioprocess = (e) => {
      if (!isRecordingRef.current) return
      const inputData = e.inputBuffer.getChannelData(0)
      recordingChunksRef.current.push(new Float32Array(inputData))
      console.log('Audio chunk collected, total chunks:', recordingChunksRef.current.length)
    }

    source.connect(processor)
    processor.connect(audioContext.destination)

    scriptProcessorRef.current = processor
    audioContextRef.current = audioContext

    setIsRecording(true)
    console.log('Recording started')
  }, [localStream])

  const stopRecording = useCallback(async () => {
    console.log('stopRecording called, isRecordingRef:', isRecordingRef.current)
    if (!isRecordingRef.current || !scriptProcessorRef.current || !audioContextRef.current) {
      console.log('stopRecording early return')
      return
    }

    const processor = scriptProcessorRef.current
    const audioContext = audioContextRef.current

    isRecordingRef.current = false
    processor.disconnect()
    audioContext.close()

    setIsRecording(false)

    await new Promise(resolve => setTimeout(resolve, 100))

    const chunks = recordingChunksRef.current
    console.log('Chunks collected:', chunks.length)
    if (chunks.length === 0) return

    const totalLength = chunks.reduce((acc, chunk) => acc + chunk.length, 0)
    const pcmData = new Float32Array(totalLength)
    let offset = 0
    for (const chunk of chunks) {
      pcmData.set(chunk, offset)
      offset += chunk.length
    }

    console.log('PCM data length:', pcmData.length)

    const int16Data = new Int16Array(pcmData.length)
    for (let i = 0; i < pcmData.length; i++) {
      const s = Math.max(-1, Math.min(1, pcmData[i]))
      int16Data[i] = s < 0 ? s * 0x8000 : s * 0x7FFF
    }

    const base64 = int16ArrayToBase64(int16Data)
    console.log('Base64 length:', base64.length)

    setAIMessages(prev => [...prev, {
      id: Date.now().toString(),
      text: '[Voice Message]',
      isUser: true,
      timestamp: new Date(),
    }])
    setIsAITyping(true)
    signalingClient.sendAIVoiceChat(base64, 16000)
    console.log('Sent ai_voice_chat')

    recordingChunksRef.current = []
    scriptProcessorRef.current = null
    audioContextRef.current = null
  }, [])

  const arrayBufferToBase64 = (buffer: ArrayBuffer): string => {
    const bytes = new Uint8Array(buffer)
    let binary = ''
    for (let i = 0; i < bytes.byteLength; i++) {
      binary += String.fromCharCode(bytes[i])
    }
    return btoa(binary)
  }

  const int16ArrayToBase64 = (int16Array: Int16Array): string => {
    const bytes = new Uint8Array(int16Array.buffer)
    let binary = ''
    for (let i = 0; i < bytes.length; i++) {
      binary += String.fromCharCode(bytes[i])
    }
    return btoa(binary)
  }

  const createWavBlob = (int16Array: Int16Array, sampleRate: number = 16000): Blob => {
    const buffer = new ArrayBuffer(44 + int16Array.length * 2)
    const view = new DataView(buffer)

    const writeString = (offset: number, str: string) => {
      for (let i = 0; i < str.length; i++) {
        view.setUint8(offset + i, str.charCodeAt(i))
      }
    }

    writeString(0, 'RIFF')
    view.setUint32(4, 36 + int16Array.length * 2, true)
    writeString(8, 'WAVE')
    writeString(12, 'fmt ')
    view.setUint32(16, 16, true)
    view.setUint16(20, 1, true)
    view.setUint16(22, 1, true)
    view.setUint32(24, sampleRate, true)
    view.setUint32(28, sampleRate * 2, true)
    view.setUint16(32, 2, true)
    view.setUint16(34, 16, true)
    writeString(36, 'data')
    view.setUint32(40, int16Array.length * 2, true)

    const dataOffset = 44
    for (let i = 0; i < int16Array.length; i++) {
      view.setInt16(dataOffset + i * 2, int16Array[i], true)
    }

    return new Blob([buffer], { type: 'audio/wav' })
  }

  const base64ToInt16Array = (base64: string): Int16Array => {
    const binary = atob(base64)
    const bytes = new Uint8Array(binary.length)
    for (let i = 0; i < binary.length; i++) {
      bytes[i] = binary.charCodeAt(i)
    }
    return new Int16Array(bytes.buffer)
  }

  const handleLeave = async () => {
    leaveRoom()
    signalingClient.disconnect()
    setCurrentRoom(null)
    onLeave()
  }

  return (
    <div className="min-h-screen bg-gray-900 flex flex-col">
      <header className="bg-gray-800 border-b border-gray-700 px-4 py-3">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-bold text-white">{currentRoom?.name}</h1>
            <p className="text-gray-400 text-sm">
              {members.length + 1} participant(s)
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

      <main className="flex-1 p-4">
        <div className="max-w-4xl mx-auto">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
            <div className="bg-gray-800 rounded-lg p-6 border border-gray-700">
              <div className="flex items-center gap-4">
                <div className={`w-12 h-12 rounded-full flex items-center justify-center ${
                  isSpeaking ? 'bg-green-500' : 'bg-gray-600'
                }`}>
                  <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z" />
                  </svg>
                </div>
                <div>
                  <p className="text-white font-medium">You</p>
                  <p className="text-gray-400 text-sm">
                    {connectionState === 'connected' ? 'Connected' : connectionState}
                  </p>
                </div>
                <button
                  onClick={toggleMute}
                  className={`ml-auto p-3 rounded-full ${
                    isMuted ? 'bg-red-500' : 'bg-gray-600'
                  } hover:opacity-80 transition-opacity`}
                >
                  {isMuted ? (
                    <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z" />
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2" />
                    </svg>
                  ) : (
                    <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z" />
                    </svg>
                  )}
                </button>
              </div>
            </div>

            {Array.from(remoteStreams.entries()).map(([userId]) => (
              <div key={userId} className="bg-gray-800 rounded-lg p-6 border border-gray-700">
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 rounded-full bg-blue-500 flex items-center justify-center">
                    <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                    </svg>
                  </div>
                  <div>
                    <p className="text-white font-medium">User {userId.slice(0, 8)}</p>
                    <p className="text-gray-400 text-sm">Connected</p>
                  </div>
                </div>
              </div>
            ))}
          </div>

          <div className="bg-gray-800 rounded-lg p-6 border border-gray-700">
            <h3 className="text-lg font-semibold text-white mb-4">Voice Activity</h3>
            <div className="flex items-center gap-2">
              {members.map((member) => (
                <div key={member.id} className="text-gray-400 text-sm">
                  {member.user_id.slice(0, 8)}...
                </div>
              ))}
            </div>
            <div className="mt-4 h-2 bg-gray-700 rounded-full overflow-hidden">
              <div
                className={`h-full transition-all duration-100 ${
                  isSpeaking ? 'bg-green-500 w-full' : 'bg-gray-600 w-1/4'
                }`}
              />
            </div>
          </div>

          <div className="bg-gray-800 rounded-lg p-6 border border-gray-700">
            <h3 className="text-lg font-semibold text-white mb-4">AI Assistant</h3>
            <div className="space-y-3 max-h-60 overflow-y-auto mb-4">
              {aiMessages.length === 0 && (
                <p className="text-gray-400 text-sm">Send a message or hold to record voice</p>
              )}
              {aiMessages.map((msg) => (
                <div key={msg.id} className={`flex ${msg.isUser ? 'justify-end' : 'justify-start'}`}>
                  <div className={`max-w-[80%] rounded-lg px-4 py-2 ${
                    msg.isUser ? 'bg-blue-600 text-white' : 'bg-gray-700 text-white'
                  }`}>
                    <p className="text-sm">{msg.text}</p>
                    {msg.isUser && msg.audio && (
                      <p className="text-xs opacity-70 mt-1">Voice message sent</p>
                    )}
                  </div>
                </div>
              ))}
              {isAITyping && (
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
                placeholder="Type a message to AI..."
                className="flex-1 bg-gray-700 text-white rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
                disabled={isAITyping}
              />
              <button
                type="submit"
                disabled={!aiInput.trim() || isAITyping}
                className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg font-medium transition-colors disabled:opacity-50"
              >
                Send
              </button>
            </form>
            <div className="mt-3 flex justify-center">
              <button
                onMouseDown={startRecording}
                onMouseUp={stopRecording}
                onTouchStart={startRecording}
                onTouchEnd={stopRecording}
                disabled={isAITyping}
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
              <p className="text-gray-400 text-sm ml-2 flex items-center">
                {isRecording ? 'Recording...' : 'Hold to speak'}
              </p>
              {isAITyping && (
                <button
                  onClick={() => signalingClient.sendInterrupt()}
                  className="ml-4 p-4 rounded-full bg-red-600 hover:bg-red-700 transition-colors"
                >
                  <svg className="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" />
                  </svg>
                </button>
              )}
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}
