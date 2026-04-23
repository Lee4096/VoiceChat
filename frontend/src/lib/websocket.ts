import type { SignalingMessage } from '../types'

type MessageHandler = (message: SignalingMessage) => void

export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting' | 'error'

interface SignalingClientOptions {
  heartbeatInterval?: number
  heartbeatTimeout?: number
  maxReconnectAttempts?: number
  baseReconnectDelay?: number
  maxReconnectDelay?: number
}

class SignalingClient {
  private ws: WebSocket | null = null
  private url: string
  private handlers: Map<string, MessageHandler[]> = new Map()
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private baseReconnectDelay = 1000
  private maxReconnectDelay = 30000
  private heartbeatInterval = 30000
  private heartbeatTimeout = 5000
  private heartbeatTimer: number | null = null
  private heartbeatTimeoutTimer: number | null = null
  private reconnectTimer: number | null = null
  private isIntentionalClose = false
  private state: ConnectionState = 'disconnected'
  private stateListeners: Set<(state: ConnectionState) => void> = new Set()
  private pingSequence = 0

  constructor(options: SignalingClientOptions = {}) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    this.url = `${protocol}//${window.location.host}/ws`

    if (options.heartbeatInterval) this.heartbeatInterval = options.heartbeatInterval
    if (options.heartbeatTimeout) this.heartbeatTimeout = options.heartbeatTimeout
    if (options.maxReconnectAttempts) this.maxReconnectAttempts = options.maxReconnectAttempts
    if (options.baseReconnectDelay) this.baseReconnectDelay = options.baseReconnectDelay
    if (options.maxReconnectDelay) this.maxReconnectDelay = options.maxReconnectDelay
  }

  private setState(newState: ConnectionState) {
    this.state = newState
    this.stateListeners.forEach(listener => listener(newState))
  }

  onStateChange(listener: (state: ConnectionState) => void) {
    this.stateListeners.add(listener)
    return () => this.stateListeners.delete(listener)
  }

  connect() {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      return Promise.resolve()
    }

    this.isIntentionalClose = false
    this.setState('connecting')

    return new Promise<void>((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.url)

        this.ws.onopen = () => {
          console.log('WebSocket connected')
          this.reconnectAttempts = 0
          this.setState('connected')
          this.startHeartbeat()
          resolve()
        }

        this.ws.onclose = (event) => {
          console.log('WebSocket disconnected', event.code, event.reason)
          this.stopHeartbeat()

          if (!this.isIntentionalClose) {
            this.handleReconnect()
          } else {
            this.setState('disconnected')
          }
        }

        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error)
          this.setState('error')
          reject(error)
        }

        this.ws.onmessage = (event) => {
          try {
            const message: SignalingMessage = JSON.parse(event.data)
            this.handlePong(message)
            this.handleMessage(message)
          } catch (e) {
            console.error('Failed to parse message:', e)
          }
        }
      } catch (error) {
        this.setState('error')
        reject(error)
      }
    })
  }

  disconnect() {
    this.isIntentionalClose = true
    this.stopHeartbeat()
    this.clearReconnectTimer()

    if (this.ws) {
      this.ws.close(1000, 'Client disconnect')
      this.ws = null
    }

    this.setState('disconnected')
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

  private handlePong(message: SignalingMessage) {
    if (message.type === 'pong') {
      this.clearHeartbeatTimeout()
      this.startHeartbeat()
    }
  }

  private startHeartbeat() {
    this.stopHeartbeat()

    this.heartbeatTimer = window.setTimeout(() => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.pingSequence++
        this.send({
          type: 'ping',
          payload: { seq: this.pingSequence },
        })

        this.heartbeatTimeoutTimer = window.setTimeout(() => {
          console.warn('Heartbeat timeout, closing connection')
          this.ws?.close(4000, 'Heartbeat timeout')
        }, this.heartbeatTimeout)
      }
    }, this.heartbeatInterval)
  }

  private stopHeartbeat() {
    if (this.heartbeatTimer) {
      clearTimeout(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
    this.clearHeartbeatTimeout()
  }

  private clearHeartbeatTimeout() {
    if (this.heartbeatTimeoutTimer) {
      clearTimeout(this.heartbeatTimeoutTimer)
      this.heartbeatTimeoutTimer = null
    }
  }

  private clearReconnectTimer() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }

  private handleReconnect() {
    if (this.isIntentionalClose) {
      return
    }

    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnect attempts reached')
      this.setState('error')
      return
    }

    this.reconnectAttempts++
    const delay = Math.min(
      this.baseReconnectDelay * Math.pow(2, this.reconnectAttempts - 1),
      this.maxReconnectDelay
    )

    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})...`)
    this.setState('reconnecting')

    this.clearReconnectTimer()
    this.reconnectTimer = window.setTimeout(() => {
      this.connect().catch(() => {
        // Will trigger handleReconnect again on failure
      })
    }, delay)
  }

  getState() {
    return this.state
  }

  isConnected() {
    return this.ws && this.ws.readyState === WebSocket.OPEN
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

  sendInterrupt() {
    this.send({ type: 'interrupt' })
  }
}

export const signalingClient = new SignalingClient()
export { SignalingClient }
