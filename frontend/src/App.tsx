import { useEffect, useState } from 'react'
import { useAuthStore } from './store/auth'
import { useRoomStore } from './store/room'
import { api } from './lib/api'
import { LoginPage } from './components/LoginPage'
import { RoomList } from './components/RoomList'
import { ChatRoom } from './components/ChatRoom'
import type { Room } from './types'

function App() {
  const { isAuthenticated, setAuth } = useAuthStore()
  const { setCurrentRoom, currentRoom } = useRoomStore()
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const initAuth = async () => {
      const params = new URLSearchParams(window.location.search)
      const provider = params.get('provider')
      const code = params.get('code')

      if (provider && code) {
        try {
          const result = await api.auth.callback(provider, code)
          if (result) {
            setAuth(result.token, result.user)
            window.history.replaceState({}, '', '/')
          }
        } catch (e) {
          console.error('Auth callback failed:', e)
        }
      }

      const token = localStorage.getItem('token')
      if (token) {
        try {
          const user = await api.users.me()
          if (user) {
            setAuth(token, user)
          }
        } catch (e) {
          console.error('Failed to fetch user:', e)
          localStorage.removeItem('token')
          localStorage.removeItem('user')
        }
      }

      setLoading(false)
    }

    initAuth()
  }, [setAuth])

  const handleJoinRoom = (room: Room) => {
    setCurrentRoom(room)
  }

  const handleLeaveRoom = () => {
    setCurrentRoom(null)
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-900">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return <LoginPage />
  }

  if (currentRoom) {
    return <ChatRoom onLeave={handleLeaveRoom} />
  }

  return <RoomList onJoinRoom={handleJoinRoom} />
}

export default App
