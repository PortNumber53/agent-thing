import { useCallback, useEffect, useRef, useState } from 'react'
import './TerminalPane.css'
import { loadLibtmtWasm, writeString, type LibtmtInstance } from '../terminal/libtmt/libtmt'

type CanvasTerminalPaneProps = {
  wsUrl: string
  isActive: boolean
  onActiveChange?: (active: boolean) => void
  rows?: number
  cols?: number
}

const ANSI_COLORS: Record<number, string> = {
  0: '#d0d0d0', // default fg
  1: '#000000',
  2: '#cc241d',
  3: '#98971a',
  4: '#d79921',
  5: '#458588',
  6: '#b16286',
  7: '#689d6a',
  8: '#a89984',
}

export function CanvasTerminalPane({
  wsUrl,
  isActive,
  onActiveChange,
  rows: initialRows = 24,
  cols: initialCols = 80,
}: CanvasTerminalPaneProps) {
  const [status, setStatus] = useState<'idle' | 'loading' | 'connecting' | 'open' | 'closed' | 'error'>('idle')
  const [loadError, setLoadError] = useState<string | null>(null)
  // Cursor blink state lives in a ref to avoid re-rendering (which would recreate callbacks).
  const cursorVisibleRef = useRef(true)
  const wsRef = useRef<WebSocket | null>(null)
  const canvasRef = useRef<HTMLCanvasElement | null>(null)
  const containerRef = useRef<HTMLDivElement | null>(null)
  const wasmRef = useRef<LibtmtInstance | null>(null)
  const vtRef = useRef<number>(0)

  const sizeRef = useRef<{ rows: number; cols: number; cellW: number; cellH: number; fontSize: number; fontFamily: string }>({
    rows: initialRows,
    cols: initialCols,
    cellW: 9,
    cellH: 18,
    fontSize: 14,
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
  })

  const buffersRef = useRef<{
    charsPtr: number
    attrsPtr: number
    chars: Uint32Array
    attrs: Uint8Array
    maxCells: number
    cursorRowPtr: number
    cursorColPtr: number
    cursorRowView: Uint32Array
    cursorColView: Uint32Array
  } | null>(null)

  const allocBuffers = useCallback((wasm: LibtmtInstance, rows: number, cols: number) => {
    const maxCells = rows * cols
    const charsPtr = wasm.malloc(maxCells * 4)
    const attrsPtr = wasm.malloc(maxCells * 3)
    const cursorRowPtr = wasm.malloc(4)
    const cursorColPtr = wasm.malloc(4)
    buffersRef.current = {
      charsPtr,
      attrsPtr,
      chars: new Uint32Array(wasm.memory.buffer, charsPtr, maxCells),
      attrs: new Uint8Array(wasm.memory.buffer, attrsPtr, maxCells * 3),
      maxCells,
      cursorRowPtr,
      cursorColPtr,
      cursorRowView: new Uint32Array(wasm.memory.buffer, cursorRowPtr, 1),
      cursorColView: new Uint32Array(wasm.memory.buffer, cursorColPtr, 1),
    }
  }, [])

  const freeBuffers = useCallback((wasm: LibtmtInstance) => {
    const buf = buffersRef.current
    if (!buf) return
    wasm.free(buf.charsPtr)
    wasm.free(buf.attrsPtr)
    wasm.free(buf.cursorRowPtr)
    wasm.free(buf.cursorColPtr)
    buffersRef.current = null
  }, [])

  const initWasm = useCallback(async () => {
    setStatus('loading')
    setLoadError(null)
    const wasmUrl = (import.meta.env.VITE_LIBTMT_WASM_URL as string | undefined) || '/libtmt.wasm'
    try {
      const wasm = await loadLibtmtWasm(wasmUrl)
      wasmRef.current = wasm
      vtRef.current = wasm.tmtw_open(sizeRef.current.rows, sizeRef.current.cols)
      allocBuffers(wasm, sizeRef.current.rows, sizeRef.current.cols)
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      setLoadError(msg)
      setStatus('error')
      throw err
    }
  }, [allocBuffers])

  const draw = useCallback(() => {
    const canvas = canvasRef.current
    const wasm = wasmRef.current
    const buf = buffersRef.current
    if (!canvas || !wasm || !buf || !vtRef.current) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    const { rows, cols, cellW, cellH, fontSize, fontFamily } = sizeRef.current
    ctx.font = `${fontSize}px ${fontFamily}`
    const width = cols * cellW
    const height = rows * cellH
    if (canvas.width !== width) canvas.width = width
    if (canvas.height !== height) canvas.height = height

    // Dump screen from wasm to buffers.
    wasm.tmtw_dump(vtRef.current, buf.charsPtr, buf.attrsPtr, buf.maxCells)

    ctx.fillStyle = '#0b0d10'
    ctx.fillRect(0, 0, width, height)

    for (let r = 0; r < rows; r++) {
      for (let c = 0; c < cols; c++) {
        const idx = r * cols + c
        const codepoint = buf.chars[idx]
        const flags = buf.attrs[idx * 3]
        const fg = buf.attrs[idx * 3 + 1]
        const bg = buf.attrs[idx * 3 + 2]

        if (bg !== 0) {
          ctx.fillStyle = ANSI_COLORS[bg] ?? '#000000'
          ctx.fillRect(c * cellW, r * cellH, cellW, cellH)
        }

        if (codepoint === 0 || (flags & (1 << 5))) continue // empty or invisible

        const ch = String.fromCodePoint(codepoint)
        ctx.fillStyle = ANSI_COLORS[fg] ?? ANSI_COLORS[0]
        ctx.fillText(ch, c * cellW, r * cellH + fontSize)
      }
    }

    // Draw blinking cursor on top.
    wasm.tmtw_get_cursor(vtRef.current, buf.cursorRowPtr, buf.cursorColPtr)
    const curR = buf.cursorRowView[0]
    const curC = buf.cursorColView[0]
    if (cursorVisibleRef.current && curR < rows && curC < cols) {
      const idx = curR * cols + curC
      const codepoint = buf.chars[idx]
      // Cursor block background.
      ctx.save()
      ctx.globalAlpha = 0.7
      ctx.fillStyle = '#e5e7eb'
      ctx.fillRect(curC * cellW, curR * cellH, cellW, cellH)
      ctx.restore()
      // Redraw character under cursor in dark for contrast.
      if (codepoint !== 0) {
        ctx.fillStyle = '#0b0d10'
        ctx.fillText(String.fromCodePoint(codepoint), curC * cellW, curR * cellH + fontSize)
      }
    }
  }, [])

  // Keep a stable ref to draw for effects.
  const drawRef = useRef(draw)
  useEffect(() => {
    drawRef.current = draw
  }, [draw])

  // Recompute terminal size from container and resize wasm/buffers.
  const resizeToContainer = useCallback(() => {
    const container = containerRef.current
    const wasm = wasmRef.current
    if (!container || !wasm || !vtRef.current) return
    const rect = container.getBoundingClientRect()
    if (rect.width <= 0 || rect.height <= 0) return

    // Measure cell size from current font.
    const fontSize = 14
    const fontFamily = sizeRef.current.fontFamily
    const tmpCanvas = canvasRef.current
    const ctx = tmpCanvas?.getContext('2d')
    if (!ctx) return
    ctx.font = `${fontSize}px ${fontFamily}`
    const metrics = ctx.measureText('M')
    const cellW = Math.max(6, Math.ceil(metrics.width))
    const cellH = Math.max(10, Math.ceil(fontSize * 1.4))

    const newCols = Math.max(10, Math.floor(rect.width / cellW))
    const newRows = Math.max(5, Math.floor(rect.height / cellH))
    const { rows, cols } = sizeRef.current
    if (newCols === cols && newRows === rows) return

    sizeRef.current = { ...sizeRef.current, rows: newRows, cols: newCols, cellW, cellH, fontSize }
    wasm.tmtw_resize(vtRef.current, newRows, newCols)
    freeBuffers(wasm)
    allocBuffers(wasm, newRows, newCols)
    draw()
  }, [allocBuffers, freeBuffers, draw])

  useEffect(() => {
    if (!isActive || status === 'idle') return
    const ro = new ResizeObserver(() => resizeToContainer())
    if (containerRef.current) ro.observe(containerRef.current)
    // Also resize once on mount.
    resizeToContainer()
    return () => ro.disconnect()
  }, [isActive, status, resizeToContainer])

  // Blink cursor at 2Hz while open.
  useEffect(() => {
    if (status !== 'open') {
      cursorVisibleRef.current = true
      return
    }
    const id = window.setInterval(() => {
      cursorVisibleRef.current = !cursorVisibleRef.current
      drawRef.current()
    }, 500)
    return () => window.clearInterval(id)
  }, [status])

  useEffect(() => {
    if (!isActive) {
      wsRef.current?.close()
      wsRef.current = null
      setStatus('idle')
      return
    }

    let cancelled = false
    ;(async () => {
      if (!wasmRef.current) {
        try {
          await initWasm()
        } catch {
          return
        }
      }
      if (cancelled) return

      setStatus('connecting')
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws
      ws.binaryType = 'arraybuffer'

      ws.onopen = () => {
        setStatus('open')
        resizeToContainer()
        draw()
      }
      ws.onclose = () => setStatus('closed')
      ws.onerror = () => setStatus('error')
      ws.onmessage = (event) => {
        const wasm = wasmRef.current
        if (!wasm || !vtRef.current) return
        const text =
          event.data instanceof ArrayBuffer
            ? new TextDecoder().decode(event.data)
            : String(event.data)

        const { ptr, len } = writeString(wasm, text)
        wasm.tmtw_write(vtRef.current, ptr, len)
        wasm.free(ptr)
        draw()
      }
    })()

    return () => {
      cancelled = true
      wsRef.current?.close()
    }
  }, [wsUrl, isActive, initWasm, draw, resizeToContainer])

  const sendBytes = useCallback((bytes: string) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return
    wsRef.current.send(bytes)
  }, [])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLCanvasElement>) => {
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

  const clearScreen = useCallback(() => {
    const wasm = wasmRef.current
    if (!wasm || !vtRef.current) return
    // tmt_reset clears state
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const resetFn = (wasm as any).tmt_reset ?? (wasm as any)._tmt_reset
    if (typeof resetFn === 'function') resetFn(vtRef.current)
    draw()
  }, [draw])

  return (
    <section className='terminal-pane'>
      <header className='terminal-pane__header'>
        <div className='terminal-pane__title'>Shell (WASM)</div>
        <div className='terminal-pane__status'>WS: {status}</div>
        {isActive ? (
          <button onClick={() => onActiveChange?.(false)}>Disconnect</button>
        ) : (
          <button onClick={() => onActiveChange?.(true)}>Connect</button>
        )}
        <button className='terminal-pane__clear' onClick={clearScreen}>
          Clear
        </button>
      </header>
      <div ref={containerRef} className='terminal-pane__output terminal-pane__output--canvas'>
        {loadError ? (
          <div style={{ padding: '12px', color: '#f1a2a2', fontFamily: 'monospace', whiteSpace: 'pre-wrap' }}>
            Failed to load libtmt WASM.
            {'\n'}
            {loadError}
            {'\n\n'}
            Build the module with Emscripten and place it at `frontend/public/libtmt.wasm`,
            or set `VITE_LIBTMT_WASM_URL` to its location.
          </div>
        ) : (
          <canvas
            ref={canvasRef}
            tabIndex={0}
            onKeyDown={handleKeyDown}
            onClick={() => canvasRef.current?.focus()}
          />
        )}
      </div>
    </section>
  )
}

export default CanvasTerminalPane
