import { useCallback, useEffect, useRef, useState } from 'react'
import './TerminalPane.css'

type TerminalPaneProps = {
  wsUrl: string
  isActive: boolean
  onActiveChange?: (active: boolean) => void
}

export function TerminalPane({ wsUrl, isActive, onActiveChange }: TerminalPaneProps) {
  const [status, setStatus] = useState<'idle' | 'connecting' | 'open' | 'closed' | 'error'>('idle')
  const [display, setDisplay] = useState<string>('')
  const [cursorPos, setCursorPos] = useState<{ row: number; col: number }>({ row: 0, col: 0 })
  const wsRef = useRef<WebSocket | null>(null)
  const outputRef = useRef<HTMLPreElement | null>(null)

  // Minimal terminal emulation state
  const emuRef = useRef<{ lines: string[]; row: number; col: number; escBuf: string }>(
    { lines: [''], row: 0, col: 0, escBuf: '' },
  )

  const applyChunk = useCallback((chunk: string) => {
    const state = emuRef.current

    const ensureLine = (r: number) => {
      while (state.lines.length <= r) state.lines.push('')
    }

    const insertChar = (ch: string) => {
      // Overwrite mode by default, matching typical terminal behavior (e.g. readline).
      // If writing beyond current line length, pad with spaces up to col.
      ensureLine(state.row)
      let line = state.lines[state.row]
      if (state.col > line.length) {
        line = line.padEnd(state.col, ' ')
      }
      const before = line.slice(0, state.col)
      const after = state.col < line.length ? line.slice(state.col + 1) : ''
      state.lines[state.row] = before + ch + after
      state.col += 1
    }

    const deleteBackspace = () => {
      ensureLine(state.row)
      if (state.col > 0) {
        const line = state.lines[state.row]
        state.lines[state.row] = line.slice(0, state.col - 1) + line.slice(state.col)
        state.col -= 1
      }
    }

    const carriageReturn = () => {
      // Many interactive programs (readline) use CR to start a full line redraw.
      // Clearing the line here keeps us in sync with those redraws.
      ensureLine(state.row)
      state.lines[state.row] = ''
      state.col = 0
    }

    const newLine = () => {
      state.row += 1
      state.col = 0
      ensureLine(state.row)
    }

    const eraseToEndOfLine = () => {
      ensureLine(state.row)
      const line = state.lines[state.row]
      state.lines[state.row] = line.slice(0, state.col)
    }

    const moveCol = (delta: number) => {
      ensureLine(state.row)
      state.col = Math.max(0, state.col + delta)
      // Clamp to current line length to avoid drifting past EOL on redraws.
      const lineLen = state.lines[state.row]?.length ?? 0
      if (state.col > lineLen) state.col = lineLen
    }

    const handleEscapeSequence = (seq: string) => {
      // Only handle CSI sequences (\x1b[ ...).
      if (!seq.startsWith('\x1b[')) return
      const body = seq.slice(2)
      const match = body.match(/^([0-9;]*)([A-Za-z~])$/)
      if (!match) return
      const paramsRaw = match[1]
      const cmd = match[2]
      const params = paramsRaw
        ? paramsRaw.split(';').map((p) => (p === '' ? 0 : Number(p)))
        : []
      const n = params[0] || 1

      switch (cmd) {
        case 'D': // cursor left
          moveCol(-n)
          break
        case 'C': // cursor right
          moveCol(n)
          break
        case 'G': // cursor horizontal absolute (1-based column)
          ensureLine(state.row)
          state.col = Math.max(0, (params[0] || 1) - 1)
          if (state.col > (state.lines[state.row]?.length ?? 0)) {
            state.col = state.lines[state.row]?.length ?? 0
          }
          break
        case 'H': // cursor position (row;col) 1-based
        case 'f':
          {
            const rowParam = params[0] || 1
            const colParam = params[1] || 1
            state.row = Math.max(0, rowParam - 1)
            ensureLine(state.row)
            state.col = Math.max(0, colParam - 1)
            const len = state.lines[state.row]?.length ?? 0
            if (state.col > len) state.col = len
          }
          break
        case 'A': // cursor up
          state.row = Math.max(0, state.row - n)
          ensureLine(state.row)
          if (state.col > (state.lines[state.row]?.length ?? 0)) {
            state.col = state.lines[state.row]?.length ?? 0
          }
          break
        case 'B': // cursor down
          state.row = state.row + n
          ensureLine(state.row)
          if (state.col > (state.lines[state.row]?.length ?? 0)) {
            state.col = state.lines[state.row]?.length ?? 0
          }
          break
        case 'K': // erase in line
          // 0K (default) erase to end
          eraseToEndOfLine()
          break
        case 'J': // erase display (we'll treat as clear screen when param=2)
          if ((params[0] || 0) === 2) {
            state.lines = ['']
            state.row = 0
            state.col = 0
          }
          break
        case 'P': // delete chars
          ensureLine(state.row)
          state.lines[state.row] =
            state.lines[state.row].slice(0, state.col) +
            state.lines[state.row].slice(state.col + n)
          break
        case '~':
          // ignore other tilde sequences
          break
        default:
          break
      }
    }

    for (let i = 0; i < chunk.length; i++) {
      const ch = chunk[i]

      // If we're collecting an escape sequence, keep buffering.
      if (state.escBuf) {
        state.escBuf += ch
        // Heuristic end: letter or ~
        if (/[A-Za-z~]/.test(ch)) {
          handleEscapeSequence(state.escBuf)
          state.escBuf = ''
        }
        continue
      }

      if (ch === '\x1b') {
        state.escBuf = '\x1b'
        continue
      }

      switch (ch) {
        case '\r':
          carriageReturn()
          break
        case '\n':
          newLine()
          break
        case '\b':
          // Many shells use BS (0x08) to move the cursor left without deleting.
          moveCol(-1)
          break
        case '\x7f':
          // DEL (0x7f) indicates a real backspace/delete.
          deleteBackspace()
          break
        default:
          insertChar(ch)
      }
    }

    // Trim buffer to last 2000 lines.
    if (state.lines.length > 2000) {
      state.lines = state.lines.slice(-2000)
      state.row = state.lines.length - 1
      state.col = Math.min(state.col, state.lines[state.row].length)
    }

    setDisplay(state.lines.join('\n'))
    setCursorPos({ row: state.row, col: state.col })
  }, [])

  useEffect(() => {
    if (!isActive) {
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }
      setStatus('idle')
      return
    }

    setStatus('connecting')
    setDisplay('')
    emuRef.current = { lines: [''], row: 0, col: 0, escBuf: '' }

    const ws = new WebSocket(wsUrl)
    wsRef.current = ws
    ws.binaryType = 'arraybuffer'

    ws.onopen = () => {
      setStatus('open')
      applyChunk('\n[connected]\n')
    }
    ws.onclose = () => {
      setStatus('closed')
      applyChunk('\n[disconnected]\n')
    }
    ws.onerror = () => {
      setStatus('error')
      applyChunk('\n[error]\n')
    }
    ws.onmessage = (event) => {
      if (event.data instanceof ArrayBuffer) {
        const text = new TextDecoder().decode(event.data)
        applyChunk(text)
      } else {
        applyChunk(String(event.data))
      }
    }

    return () => {
      ws.close()
    }
  }, [wsUrl, isActive, applyChunk])

  useEffect(() => {
    if (outputRef.current) {
      outputRef.current.scrollTop = outputRef.current.scrollHeight
    }
  }, [display])

  const sendBytes = useCallback((bytes: string) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return
    wsRef.current.send(bytes)
  }, [])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLPreElement>) => {
      if (!isActive || status !== 'open') return
      if (e.metaKey) return

      let payload: string | null = null

      if (e.ctrlKey && e.key.length === 1) {
        const upper = e.key.toUpperCase()
        const code = upper.charCodeAt(0) - 64
        if (code > 0 && code < 32) payload = String.fromCharCode(code)
      } else {
        switch (e.key) {
          case 'Enter':
            payload = '\r'
            break
          case 'Backspace':
            payload = '\x7f'
            break
          case 'Delete':
            payload = '\x1b[3~'
            break
          case 'Tab':
            payload = '\t'
            break
          case 'ArrowUp':
            payload = '\x1b[A'
            break
          case 'ArrowDown':
            payload = '\x1b[B'
            break
          case 'ArrowRight':
            payload = '\x1b[C'
            break
          case 'ArrowLeft':
            payload = '\x1b[D'
            break
          default:
            if (e.key.length === 1) payload = e.key
        }
      }

      if (payload != null) {
        e.preventDefault()
        e.stopPropagation()
        sendBytes(payload)
      }
    },
    [isActive, status, sendBytes],
  )

  const handlePaste = useCallback(
    (e: React.ClipboardEvent<HTMLPreElement>) => {
      if (!isActive || status !== 'open') return
      const text = e.clipboardData.getData('text')
      if (text) {
        e.preventDefault()
        sendBytes(text)
      }
    },
    [isActive, status, sendBytes],
  )

  return (
    <section className='terminal-pane'>
      <header className='terminal-pane__header'>
        <div className='terminal-pane__title'>Shell</div>
        <div className='terminal-pane__status'>WS: {status}</div>
        {isActive ? (
          <button onClick={() => onActiveChange?.(false)}>Disconnect</button>
        ) : (
          <button onClick={() => onActiveChange?.(true)}>Connect</button>
        )}
        <button
          className='terminal-pane__clear'
          onClick={() => setDisplay('')}
          disabled={!display}
        >
          Clear
        </button>
      </header>
      <pre
        ref={outputRef}
        className='terminal-pane__output'
        tabIndex={0}
        onKeyDown={handleKeyDown}
        onPaste={handlePaste}
        onClick={() => outputRef.current?.focus()}
      >
        {(() => {
          const baseText =
            display || (isActive ? 'Connecting to container…' : 'Click “Open shell” to connect.')
          if (!(isActive && status === 'open')) return baseText

          const marker = '\u0000'
          const lines = baseText.split('\n')
          const row = Math.max(0, Math.min(cursorPos.row, lines.length - 1))
          const line = lines[row] ?? ''
          const col = Math.max(0, Math.min(cursorPos.col, line.length))
          lines[row] = line.slice(0, col) + marker + line.slice(col)
          const withMarker = lines.join('\n')
          const parts = withMarker.split(marker)
          if (parts.length === 1) return withMarker
          return (
            <>
              {parts[0]}
              <span className='terminal-pane__cursor' aria-hidden='true'>▊</span>
              {parts.slice(1).join(marker)}
            </>
          )
        })()}
      </pre>
    </section>
  )
}

export default TerminalPane
