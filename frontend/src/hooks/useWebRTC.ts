import { useCallback, useEffect, useRef } from 'react'
import SimplePeer from 'simple-peer'
import { signalingClient } from '../lib/websocket'
import { useRoomStore } from '../store/room'
import { useAuthStore } from '../store/auth'

interface UseWebRTCOptions {
  onLocalStream?: (stream: MediaStream) => void
  onRemoteStream?: (userId: string, stream: MediaStream) => void
  onError?: (error: Error) => void
}

export function useWebRTC(options: UseWebRTCOptions = {}) {
  const peers = useRef<Map<string, SimplePeer.Instance>>(new Map())
  const localStreamRef = useRef<MediaStream | null>(null)

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

    peers.current.set(userId, peer)
    return peer
  }, [currentRoom, addRemoteStream, removeRemoteStream, options])

  const handleOffer = useCallback((userId: string, sdp: RTCSessionDescriptionInit) => {
    let peer = peers.current.get(userId)
    if (!peer) {
      const newPeer = createPeer(userId, false)
      if (newPeer) {
        newPeer.signal(sdp)
      }
    } else {
      peer.signal(sdp)
    }
  }, [createPeer])

  const handleAnswer = useCallback((userId: string, sdp: RTCSessionDescriptionInit) => {
    const peer = peers.current.get(userId)
    if (peer) {
      peer.signal(sdp)
    }
  }, [])

  const handleIceCandidate = useCallback((userId: string, candidate: RTCIceCandidateInit) => {
    const peer = peers.current.get(userId)
    if (peer) {
      peer.signal({ candidate } as SimplePeer.SignalData)
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
        const peer = peers.current.get(msg.user_id)
        if (peer) {
          peer.destroy()
          peers.current.delete(msg.user_id)
          removeRemoteStream(msg.user_id)
        }
      }
    })

    signalingClient.joinRoom(currentRoom.id, user.id, token)
  }, [currentRoom, user, token, handleOffer, handleAnswer, handleIceCandidate, createPeer, removeRemoteStream])

  const leaveRoom = useCallback(() => {
    signalingClient.leaveRoom()

    peers.current.forEach((peer) => {
      peer.destroy()
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
  }
}
