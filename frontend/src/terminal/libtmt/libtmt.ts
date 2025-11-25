// TypeScript loader and thin wrapper around libtmt WASM shim (tmt_wasm.c).

export type LibtmtInstance = {
  memory: WebAssembly.Memory
  tmtw_open: (rows: number, cols: number) => number
  tmtw_close: (vtPtr: number) => void
  tmtw_write: (vtPtr: number, strPtr: number, len: number) => void
  tmtw_resize: (vtPtr: number, rows: number, cols: number) => number
  tmtw_dump: (vtPtr: number, charsPtr: number, attrsPtr: number, maxCells: number) => number
  tmtw_get_cursor: (vtPtr: number, rowPtr: number, colPtr: number) => void
  // Standard exports from emcc STANDALONE_WASM=1:
  malloc: (n: number) => number
  free: (ptr: number) => void
}

export async function loadLibtmtWasm(url = '/libtmt.wasm'): Promise<LibtmtInstance> {
  const resp = await fetch(url)
  const bytes = await resp.arrayBuffer()
  const { instance } = await WebAssembly.instantiate(bytes, {})
  const e = instance.exports as unknown as Record<string, any>

  // Some builds prefix exports with '_' (emscripten). Normalize here.
  const pick = (name: string) => e[name] ?? e[`_${name}`]

  const memory = pick('memory') as WebAssembly.Memory
  if (!memory) throw new Error('libtmt wasm missing memory export')

  return {
    memory,
    tmtw_open: pick('tmtw_open'),
    tmtw_close: pick('tmtw_close'),
    tmtw_write: pick('tmtw_write'),
    tmtw_resize: pick('tmtw_resize'),
    tmtw_dump: pick('tmtw_dump'),
    tmtw_get_cursor: pick('tmtw_get_cursor'),
    malloc: pick('malloc'),
    free: pick('free'),
  } as LibtmtInstance
}

export function writeString(wasm: LibtmtInstance, s: string): { ptr: number; len: number } {
  const enc = new TextEncoder()
  const buf = enc.encode(s)
  const ptr = wasm.malloc(buf.length + 1)
  const mem = new Uint8Array(wasm.memory.buffer, ptr, buf.length + 1)
  mem.set(buf)
  mem[buf.length] = 0
  return { ptr, len: buf.length }
}


