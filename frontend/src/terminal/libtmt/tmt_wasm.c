// WASM shim for libtmt.
// This file is meant to be compiled with Emscripten into a standalone WASM module.
// It exposes a small C ABI that is easy to call from JS/TS.
//
// Build example (from frontend/):
//   emcc src/terminal/libtmt/tmt.c src/terminal/libtmt/tmt_wasm.c \
//     -O3 -s STANDALONE_WASM=1 \
//     -s EXPORTED_FUNCTIONS='["_tmtw_open","_tmtw_close","_tmtw_write","_tmtw_resize","_tmtw_dump","_tmtw_get_cursor"]' \
//     -o public/libtmt.wasm
//
// License: libtmt is BSD-style, see tmt.c/h headers.

#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include "tmt.h"

// Open without callback; we'll poll/dump screen from JS.
TMT *tmtw_open(uint32_t nline, uint32_t ncol) {
    return tmt_open((size_t)nline, (size_t)ncol, NULL, NULL, NULL);
}

void tmtw_close(TMT *vt) {
    if (vt) tmt_close(vt);
}

void tmtw_write(TMT *vt, const char *s, uint32_t n) {
    if (!vt || !s) return;
    tmt_write(vt, s, (size_t)n);
}

uint32_t tmtw_resize(TMT *vt, uint32_t nline, uint32_t ncol) {
    if (!vt) return 0;
    return tmt_resize(vt, (size_t)nline, (size_t)ncol) ? 1 : 0;
}

// Dump entire screen into flat buffers.
// out_chars: uint32_t array length >= nline*ncol, each is a Unicode codepoint.
// out_attrs: uint8_t array length >= nline*ncol*3, packed as:
//   [flags, fg, bg] per cell.
// flags bits: 0 bold,1 dim,2 underline,3 blink,4 reverse,5 invisible.
// fg/bg are tmt_color_t values mapped to uint8 (0 = default, 1..8 colors).
uint32_t tmtw_dump(TMT *vt, uint32_t *out_chars, uint8_t *out_attrs, uint32_t max_cells) {
    if (!vt || !out_chars || !out_attrs) return 0;
    const TMTSCREEN *s = tmt_screen(vt);
    if (!s) return 0;
    uint32_t total = (uint32_t)(s->nline * s->ncol);
    if (total > max_cells) total = max_cells;

    uint32_t idx = 0;
    for (size_t r = 0; r < s->nline && idx < total; r++) {
        TMTLINE *line = s->lines[r];
        for (size_t c = 0; c < s->ncol && idx < total; c++, idx++) {
            TMTCHAR ch = line->chars[c];
            out_chars[idx] = (uint32_t)ch.c;

            uint8_t flags = 0;
            if (ch.a.bold) flags |= (1 << 0);
            if (ch.a.dim) flags |= (1 << 1);
            if (ch.a.underline) flags |= (1 << 2);
            if (ch.a.blink) flags |= (1 << 3);
            if (ch.a.reverse) flags |= (1 << 4);
            if (ch.a.invisible) flags |= (1 << 5);

            uint8_t fg = (ch.a.fg == TMT_COLOR_DEFAULT) ? 0 : (uint8_t)ch.a.fg;
            uint8_t bg = (ch.a.bg == TMT_COLOR_DEFAULT) ? 0 : (uint8_t)ch.a.bg;

            out_attrs[idx * 3 + 0] = flags;
            out_attrs[idx * 3 + 1] = fg;
            out_attrs[idx * 3 + 2] = bg;
        }
    }
    return total;
}

void tmtw_get_cursor(TMT *vt, uint32_t *out_row, uint32_t *out_col) {
    if (!vt || !out_row || !out_col) return;
    const TMTPOINT *p = tmt_cursor(vt);
    if (!p) { *out_row = 0; *out_col = 0; return; }
    *out_row = (uint32_t)p->r;
    *out_col = (uint32_t)p->c;
}
