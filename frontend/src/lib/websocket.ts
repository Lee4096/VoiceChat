import type { SignalingMessage } from '../types'

type MessageHandler = (message: SignalingMessage) => void

class SignalingClient {
  private ws: WebSocket | null = null
  private url: string
  private handlers: Map<string, MessageHandler[]> = new Map()
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000

  constructor() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    this.url = `${protocol}//${window.location.host}/ws`
  }

  connect() {
    return new Promise<void>((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.url)

        this.ws.onopen = () => {
          console.log('WebSocket connected')
          this.reconnectAttempts = 0
          resolve()
        }

        this.ws.onclose = () => {
          console.log('WebSocket disconnected')
          this.handleReconnect()
        }

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error)
          reject(error)
        }

        this.ws.onmessage = (event) => {
          try {
            const message: SignalingMessage = JSON.parse(event.data)
            this.handleMessage(message)
          } catch (e) {
            console.error('Failed to parse message:', e)
          }
        }
      } catch (error) {
        reject(error)
      }
    })
  }

  disconnect() {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  send(message: SignalingMessage) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message))
    } else {
      console.warn('WebSocket not connected')
    }
  }

  on(type: string, handler: MessageHandler) {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, [])
    }
    this.handlers.get(type)!.push(handler)
  }

  off(type: string, handler: MessageHandler) {
    const handlers = this.handlers.get(type)
    if (handlers) {
      const index = handlers.indexOf(handler)
      if (index > -1) {
        handlers.splice(index, 1)
      }
    }
  }

  private handleMessage(message: SignalingMessage) {
    const handlers = this.handlers.get(message.type)
    if (handlers) {
      handlers.forEach(handler => handler(message))
    }

    const broadcastHandlers = this.handlers.get('*')
    if (broadcastHandlers) {
      broadcastHandlers.forEach(handler => handler(message))
    }
  }

  private handleReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++
      const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1)
      console.log(`Reconnecting in ${delay}ms...`)
      setTimeout(() => this.connect(), delay)
    }
  }

  joinRoom(roomId: string, userId: string, token: string) {
    this.send({
      type: 'join_room',
      room_id: roomId,
      user_id: userId,
      token,
    })
  }

  leaveRoom() {
    this.send({ type: 'leave_room' })
  }

  sendOffer(roomId: string, targetUserId: string, sdp: RTCSessionDescriptionInit) {
    this.send({
      type: 'offer',
      room_id: roomId,
      user_id: targetUserId,
      payload: sdp,
    })
  }

  sendAnswer(roomId: string, targetUserId: string, sdp: RTCSessionDescriptionInit) {
    this.send({
      type: 'answer',
      room_id: roomId,
      user_id: targetUserId,
      payload: sdp,
    })
  }

  sendIceCandidate(roomId: string, targetUserId: string, candidate: RTCIceCandidateInit) {
    this.send({
      type: 'ice_candidate',
      room_id: roomId,
      user_id: targetUserId,
      payload: candidate,
    })
  }

  sendVoiceData(roomId: string, audioData: ArrayBuffer) {
    this.send({
      type: 'voice_data',
      room_id: roomId,
      payload: audioData,
    })
  }

  sendAIVoiceChat(audioData: string, sampleRate: number = 16000) {
    this.send({
      type: 'ai_voice_chat',
      payload: {
        audio: audioData,
        sample_rate: sampleRate,
      },
    })
  }

  sendAITextChat(text: string) {
    this.send({
      type: 'ai_text_chat',
      payload: { text },
    })
  }

  isConnected() {
    return this.ws && this.ws.readyState === WebSocket.OPEN
  }
}

export const signalingClient = new SignalingClient()
