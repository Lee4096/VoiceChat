export interface User {
  id: string
  email: string
  name: string
  avatar: string
  provider: string
}

export interface Room {
  id: string
  name: string
  owner_id: string
  created_at: string
}

export interface RoomMember {
  id: string
  room_id: string
  user_id: string
  joined_at: string
}

export interface AuthState {
  token: string | null
  user: User | null
  isAuthenticated: boolean
}

export interface RoomState {
  currentRoom: Room | null
  members: RoomMember[]
}

export interface SignalingMessage {
  type: string
  room_id?: string
  user_id?: string
  token?: string
  payload?: unknown
}

export interface TranscriptMessage {
  type: 'transcript'
  text: string
  isFinal: boolean
  userId: string
}

export interface TTSEvent {
  type: 'tts_audio'
  audio: ArrayBuffer
  isFinal: boolean
}

export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting' | 'error'

export interface PingMessage {
  type: 'ping'
  payload: { seq: number }
}

export interface PongMessage {
  type: 'pong'
  payload: { seq: number }
}

export interface ThinkingMessage {
  type: 'thinking'
  payload: { status: 'recognizing' | 'generating' | 'done' | 'no_speech' }
}

export interface TextDeltaMessage {
  type: 'ai_text_delta'
  payload: { text: string }
}

export interface StopAudioMessage {
  type: 'stop_audio'
  payload: { user_id: string }
}
