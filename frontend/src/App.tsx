import { useCallback, useEffect, useMemo, useState } from 'react'
import TopNav from './components/TopNav'
import type { DockerStatus } from './components/TopNav'
import StatusFooter from './components/StatusFooter'
import TerminalPane from './components/TerminalPane'
import CanvasTerminalPane from './components/CanvasTerminalPane'
import './App.css'

function App() {
  const [currentTime, setCurrentTime] = useState<string>('Connecting...')
  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'open' | 'closed' | 'error'>('connecting')
  const [dockerFooter, setDockerFooter] = useState<{
    status: DockerStatus
    details: string
    message: string
  }>({ status: 'unknown', details: '', message: '' })
  const [isShellActive, setIsShellActive] = useState(false)

  const websocketUrl = useMemo(() => {
    const hostname = window.location.hostname
    const isLocalhost = hostname === 'localhost' || hostname === '127.0.0.1'
    if (isLocalhost) {
      return `ws://${hostname}:18711/ws`
    }

    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
    return `${protocol}://${window.location.host}/ws`
  }, [])

  useEffect(() => {
    const websocket = new WebSocket(websocketUrl)

    websocket.onopen = () => {
      setConnectionStatus('open')
    }

    websocket.onmessage = (event) => {
      setCurrentTime(String(event.data))
    }

    websocket.onerror = () => {
      setConnectionStatus('error')
    }

    websocket.onclose = () => {
      setConnectionStatus('closed')
    }

    return () => {
      websocket.close()
    }
  }, [websocketUrl])

  const handleDockerStatusChange = useCallback(
    ({ status, details, lastMessage }: { status: DockerStatus; details: string; lastMessage: string }) => {
      setDockerFooter({ status, details, message: lastMessage })
    },
    [],
  )

  const shellWsUrl = useMemo(() => {
    const hostname = window.location.hostname
    const isLocalhost = hostname === 'localhost' || hostname === '127.0.0.1'
    if (isLocalhost) {
      return `ws://${hostname}:18711/docker/shell`
    }
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
    return `${protocol}://${window.location.host}/docker/shell`
  }, [])

  return (
    <div className='app-container'>
      <TopNav
        onDockerStatusChange={handleDockerStatusChange}
        onOpenShell={() => setIsShellActive(true)}
      />
      <main className='app-content'>
        {(import.meta.env.VITE_TERMINAL_ENGINE as string | undefined) === 'wasm' ? (
          <CanvasTerminalPane
            wsUrl={shellWsUrl}
            isActive={isShellActive}
            onActiveChange={setIsShellActive}
          />
        ) : (
          <TerminalPane
            wsUrl={shellWsUrl}
            isActive={isShellActive}
            onActiveChange={setIsShellActive}
          />
        )}
      </main>
      <StatusFooter
        dockerStatus={dockerFooter.status}
        dockerDetails={dockerFooter.details}
        dockerMessage={dockerFooter.message}
        websocketStatus={connectionStatus}
        currentTime={currentTime}
      />
      </div>
  )
}

export default App
