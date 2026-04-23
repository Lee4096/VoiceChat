import { create } from 'zustand'
import type { Room, RoomMember, ConnectionState } from '../types'

interface RoomStore {
  currentRoom: Room | null
  members: RoomMember[]
  connectionState: ConnectionState
  localStream: MediaStream | null
  remoteStreams: Map<string, MediaStream>
  setCurrentRoom: (room: Room | null) => void
  setMembers: (members: RoomMember[]) => void
  addMember: (member: RoomMember) => void
  removeMember: (userId: string) => void
  setConnectionState: (state: ConnectionState) => void
  setLocalStream: (stream: MediaStream | null) => void
  addRemoteStream: (userId: string, stream: MediaStream) => void
  removeRemoteStream: (userId: string) => void
}

export const useRoomStore = create<RoomStore>((set) => ({
  currentRoom: null,
  members: [],
  connectionState: 'disconnected',
  localStream: null,
  remoteStreams: new Map(),
  setCurrentRoom: (room) => set({ currentRoom: room }),
  setMembers: (members) => set({ members }),
  addMember: (member) => set((state) => ({
    members: [...state.members, member]
  })),
  removeMember: (userId) => set((state) => ({
    members: state.members.filter(m => m.user_id !== userId)
  })),
  setConnectionState: (connectionState) => set({ connectionState }),
  setLocalStream: (localStream) => set({ localStream }),
  addRemoteStream: (userId, stream) => set((state) => {
    const newStreams = new Map(state.remoteStreams)
    newStreams.set(userId, stream)
    return { remoteStreams: newStreams }
  }),
  removeRemoteStream: (userId) => set((state) => {
    const newStreams = new Map(state.remoteStreams)
    newStreams.delete(userId)
    return { remoteStreams: newStreams }
  }),
}))
