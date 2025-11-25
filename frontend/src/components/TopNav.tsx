import { useCallback, useEffect, useMemo, useState } from 'react'
import './TopNav.css'

export type DockerStatus = 'unknown' | 'not_found' | 'running' | 'stopped' | 'error'

type DockerStatusResponse = {
  status: DockerStatus
  containerId?: string
  details?: string
  message?: string
}

type DockerActionResponse = {
  ok: boolean
  message: string
  status?: DockerStatus
}

type TopNavProps = {
  onDockerStatusChange?: (payload: {
    status: DockerStatus
    details: string
    lastMessage: string
  }) => void
  onOpenShell?: () => void
}

export function TopNav({ onDockerStatusChange, onOpenShell }: TopNavProps) {
  const [dockerStatus, setDockerStatus] = useState<DockerStatus>('unknown')
  const [statusDetails, setStatusDetails] = useState<string>('')
  const [isBusy, setIsBusy] = useState(false)
  const [lastMessage, setLastMessage] = useState<string>('')
  const [authToken, setAuthToken] = useState<string | null>(() => localStorage.getItem('auth_token'))
  const [isAccountOpen, setIsAccountOpen] = useState(false)

  const backendBaseUrl = useMemo(() => {
    const envBackendBaseUrl = import.meta.env.VITE_BACKEND_BASE_URL as string | undefined
    if (envBackendBaseUrl && envBackendBaseUrl.trim() !== '') {
      return envBackendBaseUrl.trim().replace(/\/+$/, '')
    }

    const hostname = window.location.hostname
    const isLocalhost = hostname === 'localhost' || hostname === '127.0.0.1'
    if (isLocalhost) {
      return `http://${hostname}:18711`
    }

    const protocol = window.location.protocol === 'https:' ? 'https' : 'http'
    return `${protocol}://${window.location.host}`
  }, [])

  const refreshStatus = useCallback(async () => {
    try {
      const response = await fetch(`${backendBaseUrl}/docker/status`)
      const data = (await response.json()) as DockerStatusResponse
      setDockerStatus(data.status)
      setStatusDetails(data.details ?? data.message ?? '')
    } catch (error) {
      setDockerStatus('error')
      setStatusDetails(String(error))
    }
  }, [backendBaseUrl])

  useEffect(() => {
    refreshStatus()
  }, [refreshStatus])

  useEffect(() => {
    onDockerStatusChange?.({
      status: dockerStatus,
      details: statusDetails,
      lastMessage,
    })
  }, [dockerStatus, statusDetails, lastMessage, onDockerStatusChange])

  // If callback returns a token as query param, store it and clean URL.
  useEffect(() => {
    if (import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.log('[auth] TopNav mounted; href=', window.location.href)
      // eslint-disable-next-line no-console
      console.log('[auth] initial localStorage auth_token len=', (localStorage.getItem('auth_token') || '').length)
    }
    const params = new URLSearchParams(window.location.search)
    const tokenFromUrl = params.get('token')
    if (tokenFromUrl) {
      if (import.meta.env.DEV) {
        // eslint-disable-next-line no-console
        console.log('[auth] token found in URL, storing to localStorage; len=', tokenFromUrl.length)
      }
      localStorage.setItem('auth_token', tokenFromUrl)
      setAuthToken(tokenFromUrl)
      params.delete('token')
      const newSearch = params.toString()
      const newUrl = `${window.location.pathname}${newSearch ? `?${newSearch}` : ''}${window.location.hash}`
      window.history.replaceState({}, '', newUrl)
    } else if (import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.log('[auth] no token in URL on load')
    }
  }, [])

  useEffect(() => {
    if (import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.log('[auth] authToken state changed; present=', Boolean(authToken), 'len=', authToken?.length ?? 0)
    }
  }, [authToken])

  const runAction = useCallback(
    async (action: 'start' | 'stop' | 'rebuild') => {
      setIsBusy(true)
      setLastMessage('')
      try {
        const response = await fetch(`${backendBaseUrl}/docker/${action}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
        })
        const data = (await response.json()) as DockerActionResponse
        setLastMessage(data.message)
      } catch (error) {
        setLastMessage(String(error))
      } finally {
        setIsBusy(false)
        await refreshStatus()
      }
    },
    [backendBaseUrl, refreshStatus],
  )

  const googleLoginUrl = `${backendBaseUrl}/auth/google/login`

  const handleLogin = () => {
    window.location.href = googleLoginUrl
  }

  const handleSignup = () => {
    window.location.href = googleLoginUrl
  }

  const handleLogout = () => {
    localStorage.removeItem('auth_token')
    setAuthToken(null)
    setIsAccountOpen(false)
  }

  return (
    <nav className='top-nav'>
      <div className='top-nav__title'>Agent Thing</div>
      <div className='top-nav__controls'>
        <button onClick={() => runAction('start')} disabled={isBusy}>
          Start
        </button>
        <button onClick={() => runAction('stop')} disabled={isBusy}>
          Stop
        </button>
        <button onClick={() => runAction('rebuild')} disabled={isBusy}>
          Rebuild
        </button>
        <button onClick={refreshStatus} disabled={isBusy}>
          Status
        </button>
        <button onClick={onOpenShell} disabled={isBusy}>
          Open shell
        </button>
      </div>
      <div className='top-nav__right'>
        <div className='top-nav__auth'>
          {!authToken ? (
            <>
              <button className='top-nav__auth-btn' onClick={handleLogin} disabled={isBusy}>
                Log in
              </button>
              <button className='top-nav__auth-btn top-nav__auth-btn--primary' onClick={handleSignup} disabled={isBusy}>
                Sign up
              </button>
            </>
          ) : (
            <div className='top-nav__account'>
              <button
                className='top-nav__auth-btn'
                onClick={() => setIsAccountOpen((v) => !v)}
                aria-expanded={isAccountOpen}
                aria-haspopup='menu'
              >
                Account ▾
              </button>
              {isAccountOpen && (
                <div className='top-nav__account-menu' role='menu'>
                  <a role='menuitem' href='/profile' className='top-nav__account-link'>
                    Profile
                  </a>
                  <a role='menuitem' href='/settings' className='top-nav__account-link'>
                    Settings
                  </a>
                  <div className='top-nav__account-separator' />
                  <button role='menuitem' onClick={handleLogout}>
                    Log out
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </nav>
  )
}

export default TopNav
