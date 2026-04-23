import { useCallback, useEffect, useRef } from 'react'
import SimplePeer from 'simple-peer'
import { signalingClient } from '../lib/websocket'
import { useRoomStore } from '../store/room'
import { useAuthStore } from '../store/auth'

interface UseWebRTCOptions {
  onLocalStream?: (stream: MediaStream) => void
  onRemoteStream?: (userId: string, stream: MediaStream) => void
  onError?: (error: Error) => void
  onReconnecting?: (attempt: number) => void
  onReconnected?: () => void
}

interface PeerConnection {
  peer: SimplePeer.Instance
  userId: string
  iceConnectionState: RTCIceConnectionState
}

export function useWebRTC(options: UseWebRTCOptions = {}) {
  const peers = useRef<Map<string, PeerConnection>>(new Map())
  const localStreamRef = useRef<MediaStream | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const maxReconnectAttempts = 10
  const baseReconnectDelay = 1000
  const maxReconnectDelay = 30000
  const reconnectTimerRef = useRef<number | null>(null)
  const isReconnectingRef = useRef(false)

  const { setLocalStream, addRemoteStream, removeRemoteStream, currentRoom } = useRoomStore()
  const { user, token } = useAuthStore()

  const getLocalStream = useCallback(async () => {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
        },
        video: false,
      })
      localStreamRef.current = stream
      setLocalStream(stream)
      options.onLocalStream?.(stream)
      return stream
    } catch (error) {
      options.onError?.(error as Error)
      throw error
    }
  }, [setLocalStream, options])

  const monitorPeerConnection = useCallback((peer: SimplePeer.Instance, userId: string) => {
    const rtcPeer = (peer as any)._pc as RTCPeerConnection | undefined
    if (!rtcPeer) return

    const checkState = () => {
      const state = rtcPeer.iceConnectionState
      const conn = peers.current.get(userId)
      if (conn) {
        conn.iceConnectionState = state
      }

      console.log(`ICE connection state for ${userId}:`, state)

      if (state === 'disconnected' || state === 'failed') {
        handleDisconnection(userId)
      } else if (state === 'connected') {
        handleSuccessfulConnection()
      }
    }

    rtcPeer.addEventListener('iceconnectionstatechange', checkState)
    checkState()

    return () => {
      rtcPeer.removeEventListener('iceconnectionstatechange', checkState)
    }
  }, [])

  const handleDisconnection = useCallback((userId: string) => {
    if (isReconnectingRef.current) return

    isReconnectingRef.current = true
    reconnectAttemptsRef.current++

    const attempt = reconnectAttemptsRef.current
    const delay = Math.min(
      baseReconnectDelay * Math.pow(2, attempt - 1),
      maxReconnectDelay
    )

    console.log(`WebRTC disconnected for ${userId}, attempt ${attempt} in ${delay}ms`)
    options.onReconnecting?.(attempt)

    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
    }

    reconnectTimerRef.current = window.setTimeout(() => {
      const conn = peers.current.get(userId)
      if (conn) {
        conn.peer.destroy()
        peers.current.delete(userId)
        removeRemoteStream(userId)
      }

      if (userId && localStreamRef.current) {
        const newPeer = createPeer(userId, true)
        if (newPeer && reconnectAttemptsRef.current < maxReconnectAttempts) {
          console.log(`Reconnection attempt ${reconnectAttemptsRef.current} for ${userId}`)
        }
      }

      isReconnectingRef.current = false
    }, delay)
  }, [options, removeRemoteStream])

  const handleSuccessfulConnection = useCallback(() => {
    console.log('WebRTC connected successfully')
    reconnectAttemptsRef.current = 0
    isReconnectingRef.current = false

    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }

    options.onReconnected?.()
  }, [options])

  const createPeer = useCallback((userId: string, initiator: boolean) => {
    if (!localStreamRef.current) {
      console.error('Local stream not available')
      return null
    }

    const peer = new SimplePeer({
      initiator,
      stream: localStreamRef.current,
      trickle: true,
    })

    peer.on('signal', (data) => {
      if (currentRoom) {
        if (data.type === 'offer') {
          signalingClient.sendOffer(currentRoom.id, userId, data)
        } else if (data.type === 'answer') {
          signalingClient.sendAnswer(currentRoom.id, userId, data)
        } else {
          signalingClient.sendIceCandidate(currentRoom.id, userId, data as RTCIceCandidateInit)
        }
      }
    })

    peer.on('stream', (remoteStream) => {
      addRemoteStream(userId, remoteStream)
      options.onRemoteStream?.(userId, remoteStream)
    })

    peer.on('close', () => {
      peers.current.delete(userId)
      removeRemoteStream(userId)
    })

    peer.on('error', (err) => {
      console.error(`Peer error for ${userId}:`, err)
      options.onError?.(err)
    })

    peers.current.set(userId, { peer, userId, iceConnectionState: 'new' })
    monitorPeerConnection(peer, userId)

    return peer
  }, [currentRoom, addRemoteStream, removeRemoteStream, options, monitorPeerConnection])

  const handleOffer = useCallback((userId: string, sdp: RTCSessionDescriptionInit) => {
    let conn = peers.current.get(userId)
    if (!conn) {
      const newPeer = createPeer(userId, false)
      if (newPeer) {
        newPeer.signal(sdp)
      }
    } else {
      conn.peer.signal(sdp)
    }
  }, [createPeer])

  const handleAnswer = useCallback((userId: string, sdp: RTCSessionDescriptionInit) => {
    const conn = peers.current.get(userId)
    if (conn) {
      conn.peer.signal(sdp)
    }
  }, [])

  const handleIceCandidate = useCallback((userId: string, candidate: RTCIceCandidateInit) => {
    const conn = peers.current.get(userId)
    if (conn) {
      conn.peer.signal({ candidate } as SimplePeer.SignalData)
    }
  }, [])

  const joinRoom = useCallback(async () => {
    if (!currentRoom || !user || !token) return

    await signalingClient.connect()

    signalingClient.on('offer', (msg) => {
      if (msg.user_id && msg.payload) {
        handleOffer(msg.user_id, msg.payload as RTCSessionDescriptionInit)
      }
    })

    signalingClient.on('answer', (msg) => {
      if (msg.user_id && msg.payload) {
        handleAnswer(msg.user_id, msg.payload as RTCSessionDescriptionInit)
      }
    })

    signalingClient.on('ice_candidate', (msg) => {
      if (msg.user_id && msg.payload) {
        handleIceCandidate(msg.user_id, msg.payload as RTCIceCandidateInit)
      }
    })

    signalingClient.on('user_joined', (msg) => {
      if (msg.user_id && msg.user_id !== user.id) {
        createPeer(msg.user_id, true)
      }
    })

    signalingClient.on('user_left', (msg) => {
      if (msg.user_id) {
        const conn = peers.current.get(msg.user_id)
        if (conn) {
          conn.peer.destroy()
          peers.current.delete(msg.user_id)
          removeRemoteStream(msg.user_id)
        }
      }
    })

    signalingClient.joinRoom(currentRoom.id, user.id, token)
  }, [currentRoom, user, token, handleOffer, handleAnswer, handleIceCandidate, createPeer, removeRemoteStream])

  const leaveRoom = useCallback(() => {
    signalingClient.leaveRoom()

    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }
    reconnectAttemptsRef.current = 0
    isReconnectingRef.current = false

    peers.current.forEach((conn) => {
      conn.peer.destroy()
    })
    peers.current.clear()

    if (localStreamRef.current) {
      localStreamRef.current.getTracks().forEach((track) => track.stop())
      localStreamRef.current = null
    }

    setLocalStream(null)
  }, [setLocalStream])

  useEffect(() => {
    return () => {
      leaveRoom()
    }
  }, [leaveRoom])

  return {
    getLocalStream,
    joinRoom,
    leaveRoom,
    createPeer,
    reconnectAttempts: reconnectAttemptsRef.current,
  }
}
