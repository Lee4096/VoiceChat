import { useState, useEffect } from 'react'
import { api } from '../lib/api'
import { useAuthStore } from '../store/auth'

interface Room {
  id: string
  name: string
  owner_id: string
  created_at: string
}

interface RoomListProps {
  onJoinRoom: (room: Room) => void
}

export function RoomList({ onJoinRoom }: RoomListProps) {
  const [rooms, setRooms] = useState<Room[]>([])
  const [loading, setLoading] = useState(true)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [newRoomName, setNewRoomName] = useState('')
  const [creating, setCreating] = useState(false)

  const { user, logout } = useAuthStore()

  useEffect(() => {
    loadRooms()
  }, [])

  const loadRooms = async () => {
    try {
      const data = await api.rooms.list()
      setRooms(data)
    } catch (e) {
      console.error('Failed to load rooms:', e)
    } finally {
      setLoading(false)
    }
  }

  const handleCreateRoom = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newRoomName.trim()) return

    setCreating(true)
    try {
      const room = await api.rooms.create(newRoomName.trim())
      if (room) {
        setRooms([{ ...room, created_at: new Date().toISOString() }, ...rooms])
        setShowCreateModal(false)
        setNewRoomName('')
      }
    } catch (e) {
      console.error('Failed to create room:', e)
    } finally {
      setCreating(false)
    }
  }

  const handleJoinRoom = async (room: Room) => {
    try {
      await api.rooms.join(room.id)
      onJoinRoom(room)
    } catch (e) {
      console.error('Failed to join room:', e)
    }
  }

  return (
    <div className="min-h-screen bg-gray-900">
      <header className="bg-gray-800 border-b border-gray-700">
        <div className="max-w-6xl mx-auto px-4 py-4 flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-white">VoiceChat</h1>
            <p className="text-gray-400 text-sm">Welcome, {user?.name}</p>
          </div>
          <div className="flex items-center gap-4">
            <button
              onClick={() => setShowCreateModal(true)}
              className="bg-blue-600 hover:bg-blue-700 text-white py-2 px-4 rounded-lg font-medium transition-colors"
            >
              Create Room
            </button>
            <button
              onClick={logout}
              className="text-gray-400 hover:text-white transition-colors"
            >
              Logout
            </button>
          </div>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-4 py-8">
        <h2 className="text-xl font-semibold text-white mb-6">Available Rooms</h2>

        {loading ? (
          <div className="flex justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
          </div>
        ) : rooms.length === 0 ? (
          <div className="bg-gray-800 rounded-lg p-8 text-center">
            <p className="text-gray-400 mb-4">No rooms available</p>
            <button
              onClick={() => setShowCreateModal(true)}
              className="text-blue-400 hover:text-blue-300 transition-colors"
            >
              Create the first room
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {rooms.map((room) => (
              <div
                key={room.id}
                className="bg-gray-800 rounded-lg p-6 border border-gray-700 hover:border-gray-600 transition-colors"
              >
                <h3 className="text-lg font-medium text-white mb-2">{room.name}</h3>
                <p className="text-gray-400 text-sm mb-4">
                  Created {new Date(room.created_at).toLocaleDateString()}
                </p>
                <button
                  onClick={() => handleJoinRoom(room)}
                  className="w-full bg-blue-600 hover:bg-blue-700 text-white py-2 px-4 rounded-lg font-medium transition-colors"
                >
                  Join Room
                </button>
              </div>
            ))}
          </div>
        )}
      </main>

      {showCreateModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-gray-800 rounded-lg p-6 w-full max-w-md mx-4">
            <h3 className="text-xl font-semibold text-white mb-4">Create Room</h3>
            <form onSubmit={handleCreateRoom}>
              <input
                type="text"
                value={newRoomName}
                onChange={(e) => setNewRoomName(e.target.value)}
                placeholder="Room name"
                className="w-full bg-gray-700 text-white rounded-lg px-4 py-3 mb-4 focus:outline-none focus:ring-2 focus:ring-blue-500"
                autoFocus
              />
              <div className="flex gap-3">
                <button
                  type="button"
                  onClick={() => {
                    setShowCreateModal(false)
                    setNewRoomName('')
                  }}
                  className="flex-1 bg-gray-700 hover:bg-gray-600 text-white py-2 px-4 rounded-lg font-medium transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={creating || !newRoomName.trim()}
                  className="flex-1 bg-blue-600 hover:bg-blue-700 text-white py-2 px-4 rounded-lg font-medium transition-colors disabled:opacity-50"
                >
                  {creating ? 'Creating...' : 'Create'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
