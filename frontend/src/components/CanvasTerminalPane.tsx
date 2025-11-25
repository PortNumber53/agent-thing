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
  const [textLayer, setTextLayer] = useState<string>('')
  const [layerStyle, setLayerStyle] = useState<{
    fontSize: number
    lineHeight: number
    fontFamily: string
  }>({
    fontSize: 14,
    lineHeight: 18,
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
  })
  // Cursor blink state lives in a ref to avoid re-rendering (which would recreate callbacks).
  const cursorVisibleRef = useRef(true)
  const wsRef = useRef<WebSocket | null>(null)
  const canvasRef = useRef<HTMLCanvasElement | null>(null)
  const containerRef = useRef<HTMLDivElement | null>(null)
  const measurerRef = useRef<HTMLPreElement | null>(null)
  const textLayerRef = useRef<HTMLPreElement | null>(null)
  const wasmRef = useRef<LibtmtInstance | null>(null)
  const vtRef = useRef<number>(0)
  const lastResizeAtRef = useRef<number>(0)
  const lastAppliedWidthRef = useRef<number>(0)
  const lastAppliedHeightRef = useRef<number>(0)
  const didPostFirstOutputResizeRef = useRef<boolean>(false)

  const sendResizeToServer = useCallback((rows: number, cols: number) => {
    const ws = wsRef.current
    if (!ws || ws.readyState !== WebSocket.OPEN) return
    try {
      ws.send(JSON.stringify({ type: 'resize', rows, cols }))
    } catch {
      // ignore
    }
  }, [])

  const sizeRef = useRef<{
    rows: number
    cols: number
    cellW: number
    cellH: number
    fontSize: number
    fontFamily: string
  }>({
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

  const buildTextLayer = useCallback((): string => {
    const wasm = wasmRef.current
    const buf = buffersRef.current
    if (!wasm || !buf || !vtRef.current) return ''
    const { rows, cols } = sizeRef.current
    wasm.tmtw_dump(vtRef.current, buf.charsPtr, buf.attrsPtr, buf.maxCells)
    const lines: string[] = []
    for (let r = 0; r < rows; r++) {
      let line = ''
      for (let c = 0; c < cols; c++) {
        const idx = r * cols + c
        const cp = buf.chars[idx]
        line += cp === 0 ? ' ' : String.fromCodePoint(cp)
      }
      lines.push(line)
    }
    return lines.join('\n')
  }, [])

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
    ctx.textBaseline = 'top'
    ctx.textAlign = 'left'

    // Size canvas to the container in CSS pixels and scale for DPR to avoid browser scaling drift.
    // The terminal grid uses cols/rows derived from this same container size, so the text layer and
    // canvas remain aligned while the terminal stays fluid.
    const containerRect = containerRef.current?.getBoundingClientRect()
    const cssW = containerRect?.width ?? cols * cellW
    const cssH = containerRect?.height ?? rows * cellH
    const dpr = window.devicePixelRatio || 1
    const pxW = Math.max(1, Math.floor(cssW * dpr))
    const pxH = Math.max(1, Math.floor(cssH * dpr))
    if (canvas.width !== pxW) canvas.width = pxW
    if (canvas.height !== pxH) canvas.height = pxH
    canvas.style.width = `${cssW}px`
    canvas.style.height = `${cssH}px`

    // Reset transform and scale to CSS pixels.
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)

    // Dump screen from wasm to buffers.
    wasm.tmtw_dump(vtRef.current, buf.charsPtr, buf.attrsPtr, buf.maxCells)

    ctx.fillStyle = '#0b0d10'
    ctx.fillRect(0, 0, cssW, cssH)

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
        ctx.fillText(ch, c * cellW, r * cellH)
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
        ctx.fillText(String.fromCodePoint(codepoint), curC * cellW, curR * cellH)
      }
    }
  }, [])

  // Keep a stable ref to draw for effects.
  const drawRef = useRef(draw)
  useEffect(() => {
    drawRef.current = draw
  }, [draw])

  // Recompute terminal size from container and resize wasm/buffers.
  // When force=true, bypass hysteresis/early-returns so we always apply a fresh PTY size
  // (useful on initial focus/connect).
  const resizeToContainer = useCallback((force = false) => {
    const container = containerRef.current
    const wasm = wasmRef.current
    if (!container || !wasm || !vtRef.current) return
    const rect = container.getBoundingClientRect()
    if (rect.width <= 0 || rect.height <= 0) return

    // Measure cell size from a DOM pre using the exact same font metrics as selection layer.
    const fontSize = sizeRef.current.fontSize
    const fontFamily = sizeRef.current.fontFamily
    let cellW = sizeRef.current.cellW
    let cellH = sizeRef.current.cellH

    const measurer = measurerRef.current
    if (measurer) {
      measurer.style.fontSize = `${fontSize}px`
      measurer.style.fontFamily = fontFamily
      measurer.style.letterSpacing = '0px'
      // Use a long run to reduce rounding error when averaging cell width.
      measurer.textContent = `${'M'.repeat(100)}\nM`
      const mrect = measurer.getBoundingClientRect()
      const lines = 2
      const colsMeasured = 100
      cellW = mrect.width / colsMeasured
      cellH = mrect.height / lines
      if (!Number.isFinite(cellW) || cellW < 6) cellW = 9
      if (!Number.isFinite(cellH) || cellH < 10) cellH = 18
    }

    // Derive columns/rows from available space.
    // Use hysteresis to avoid +/-1 jitter that can resize libtmt while typing.
    // For cols, prefer ceil when it still fits within a small tolerance. This avoids chronic
    // under-counting when DOM-measured cellW is slightly larger than actual glyph width.
    const rawCols = rect.width / cellW
    const floorCols = Math.floor(rawCols)
    const ceilCols = Math.ceil(rawCols)
    const fitsCeil = ceilCols * cellW <= rect.width + cellW * 0.2
    const targetCols = Math.max(10, fitsCeil ? ceilCols : floorCols)
    const targetRows = Math.max(5, Math.floor((rect.height + cellH * 0.1) / cellH))
    const { rows, cols } = sizeRef.current
    if (!force && targetCols === cols && targetRows === rows) return

    const now = performance.now()
    const deltaCols = targetCols - cols
    const deltaRows = targetRows - rows
    const recent = now - lastResizeAtRef.current < 80

    let newCols = targetCols
    let newRows = targetRows
    if (!force && Math.abs(deltaCols) === 1) {
      const lastW = lastAppliedWidthRef.current || rect.width
      if (Math.abs(rect.width - lastW) < cellW * 0.6 && recent) {
        newCols = cols
      }
    }
    if (!force && Math.abs(deltaRows) === 1) {
      const lastH = lastAppliedHeightRef.current || rect.height
      if (Math.abs(rect.height - lastH) < cellH * 0.6 && recent) {
        newRows = rows
      }
    }
    if (!force && newCols === cols && newRows === rows) return

    sizeRef.current = { ...sizeRef.current, rows: newRows, cols: newCols, cellW, cellH, fontSize }
    wasm.tmtw_resize(vtRef.current, newRows, newCols)
    sendResizeToServer(newRows, newCols)
    freeBuffers(wasm)
    allocBuffers(wasm, newRows, newCols)
    draw()
    setTextLayer(buildTextLayer())
    setLayerStyle({
      fontSize: sizeRef.current.fontSize,
      lineHeight: sizeRef.current.cellH,
      fontFamily: sizeRef.current.fontFamily,
    })
    lastResizeAtRef.current = now
    lastAppliedWidthRef.current = rect.width
    lastAppliedHeightRef.current = rect.height
  }, [allocBuffers, freeBuffers, draw, buildTextLayer, sendResizeToServer])

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
        // Initial layout can still be settling (fonts, scrollbars, parent flex),
        // so perform a couple of follow-up resizes to ensure PTY and wasm grid match.
        didPostFirstOutputResizeRef.current = false
        resizeToContainer(true)
        requestAnimationFrame(() => resizeToContainer(true))
        window.setTimeout(() => resizeToContainer(true), 120)
        draw()
        setTextLayer(buildTextLayer())
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

        // After the first chunk of PTY output arrives, the layout/fonts/scrollbar gutter
        // are guaranteed to be settled. Re-run resize once to ensure PTY cols/rows match
        // the final measured canvas width regardless of focus/connect ordering.
        if (!didPostFirstOutputResizeRef.current) {
          didPostFirstOutputResizeRef.current = true
          window.setTimeout(() => resizeToContainer(true), 0)
        }

        draw()
        setTextLayer(buildTextLayer())
      }
    })()

    return () => {
      cancelled = true
      wsRef.current?.close()
    }
  }, [wsUrl, isActive, initWasm, draw, resizeToContainer, buildTextLayer])

  const sendBytes = useCallback((bytes: string) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return
    wsRef.current.send(bytes)
  }, [])

  const pasteFromClipboard = useCallback(async () => {
    try {
      const text = await navigator.clipboard.readText()
      if (text) sendBytes(text)
    } catch {
      // ignore; browser may block without user gesture
    }
  }, [sendBytes])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLElement>) => {
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

  const handleKeyDownWithClipboard = useCallback(
    (e: React.KeyboardEvent<HTMLElement>) => {
      if (!isActive || status !== 'open') return

      const keyLower = e.key.toLowerCase()

      // Paste shortcuts:
      // - macOS: Cmd+V
      // - Windows/Linux: Ctrl+Shift+V
      const isPaste =
        (e.metaKey && !e.shiftKey && keyLower === 'v') ||
        (e.ctrlKey && e.shiftKey && keyLower === 'v')

      if (isPaste) {
        e.preventDefault()
        e.stopPropagation()
        void pasteFromClipboard()
        return
      }

      handleKeyDown(e)
    },
    [isActive, status, pasteFromClipboard, handleKeyDown],
  )

  const handlePaste = useCallback(
    (e: React.ClipboardEvent<HTMLElement>) => {
      if (!isActive || status !== 'open') return
      const text = e.clipboardData.getData('text')
      if (text) {
        e.preventDefault()
        e.stopPropagation()
        sendBytes(text)
      }
    },
    [isActive, status, sendBytes],
  )

  const selectLineAtClientY = useCallback(
    (clientY: number) => {
      const pre = textLayerRef.current
      if (!pre) return
      const rect = pre.getBoundingClientRect()
      const { cols, rows } = sizeRef.current
      const lineHeight = layerStyle.lineHeight || 18
      const y = clientY - rect.top
      const lineIdx = Math.max(0, Math.min(rows - 1, Math.floor(y / lineHeight)))

      const textNode = pre.firstChild
      if (!textNode || textNode.nodeType !== Node.TEXT_NODE) return

      const textLen = (textNode as Text).data.length
      const lineStart = lineIdx * (cols + 1) // +1 for '\n'
      const lineEnd = Math.min(textLen, lineStart + cols)

      const range = document.createRange()
      range.setStart(textNode, lineStart)
      range.setEnd(textNode, lineEnd)

      const sel = window.getSelection()
      if (sel) {
        sel.removeAllRanges()
        sel.addRange(range)
      }
    },
    [layerStyle.lineHeight],
  )

  const handleTextLayerMouseUp = useCallback(
    (e: React.MouseEvent<HTMLPreElement>) => {
      // detail==2 for double click, ==3 for triple click.
      if (e.detail >= 2) {
        e.preventDefault()
        e.stopPropagation()
        // Native selection is applied after this handler; apply ours on next frame.
        requestAnimationFrame(() => selectLineAtClientY(e.clientY))
      }
    },
    [selectLineAtClientY],
  )

  const clearScreen = useCallback(() => {
    const wasm = wasmRef.current
    if (!wasm || !vtRef.current) return
    // tmt_reset clears state
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const resetFn = (wasm as any).tmt_reset ?? (wasm as any)._tmt_reset
    if (typeof resetFn === 'function') resetFn(vtRef.current)
    draw()
    setTextLayer(buildTextLayer())
  }, [draw, buildTextLayer])

  const handleFocusReflow = useCallback(() => {
    if (status !== 'open') return
    // Same settle-pass as ws.onopen for cases where container width
    // changes without triggering ResizeObserver (scrollbar/gutter/font swap).
    resizeToContainer(true)
    requestAnimationFrame(() => resizeToContainer(true))
    window.setTimeout(() => resizeToContainer(true), 120)
    draw()
    setTextLayer(buildTextLayer())
  }, [status, resizeToContainer, draw, buildTextLayer])

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
          <div className='terminal-pane__canvas-wrap'>
            <canvas ref={canvasRef} />
            {/* Hidden measurer to align DOM and canvas metrics. */}
            <pre
              ref={measurerRef}
              aria-hidden='true'
              style={{
                position: 'absolute',
                visibility: 'hidden',
                whiteSpace: 'pre',
                padding: '0',
                margin: '0',
                inset: '0 auto auto 0',
              }}
            />
            <pre
              className='terminal-pane__text-layer'
              ref={textLayerRef}
              tabIndex={0}
              style={{
                fontSize: layerStyle.fontSize,
                lineHeight: `${layerStyle.lineHeight}px`,
                fontFamily: layerStyle.fontFamily,
                // Small constant tweak so DOM selection aligns with canvas glyphs.
                transform: 'translateY(-2px)',
              }}
              onKeyDown={handleKeyDownWithClipboard}
              onPaste={handlePaste}
              onFocus={handleFocusReflow}
              onMouseUp={handleTextLayerMouseUp}
            >
              {textLayer}
            </pre>
          </div>
        )}
      </div>
    </section>
  )
}

export default CanvasTerminalPane
