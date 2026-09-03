# Plan: Integrate Magpie WASM into Woogles

## Summary

Integrate the magpie WASM analysis engine into Woogles (liwords) to enable Monte Carlo simulation and endgame solving. This requires adding COOP/COEP headers site-wide and fixing all cross-origin resources.

## Architecture Decision

**Site-wide COOP/COEP headers** with CORS fixes for all external resources. This is simpler than an iframe approach and allows direct WASM integration.

---

## Phase 1: CORS and Cross-Origin Isolation Setup

### 1.1 CloudFront Response Headers Policy

Add headers to woogles.io CloudFront distribution (`aws/cfn/cloudfront.yaml`):

```yaml
# Add to DefaultCacheBehavior
ResponseHeadersPolicyId: !Ref CrossOriginIsolationPolicy

# New resource
CrossOriginIsolationPolicy:
  Type: AWS::CloudFront::ResponseHeadersPolicy
  Properties:
    ResponseHeadersPolicyConfig:
      Name: cross-origin-isolation
      SecurityHeadersConfig:
        CrossOriginOpenerPolicy:
          Override: true
          Value: same-origin
        CrossOriginEmbedderPolicy:
          Override: true
          Value: require-corp
```

### 1.2 Fix Google Fonts (index.html)

Add `crossorigin` to all font `<link>` tags (Google Fonts supports CORS):

```html
<link href="https://fonts.googleapis.com/css2?family=Mulish..." rel="stylesheet" crossorigin />
```

**File**: `liwords-ui/index.html` (lines 26-45) - 5 font families

### 1.3 Fix Font Awesome CDN

```html
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/.../all.min.css" crossorigin />
```

**File**: `liwords-ui/index.html` (line 47-50)

### 1.4 S3 Bucket CORS Configuration

**Buckets requiring CORS:**

| Bucket | Used For |
|--------|----------|
| `woogles-prod-assets` | Board textures, macondog avatar |
| `woogles-flags` | Country flag images |
| `woogles-uploads` | User avatars |

**S3 CORS Policy (each bucket):**

```json
{
  "CORSRules": [{
    "AllowedHeaders": ["*"],
    "AllowedMethods": ["GET", "HEAD"],
    "AllowedOrigins": ["https://woogles.io", "https://www.woogles.io"],
    "ExposeHeaders": ["Content-Length", "Content-Type"],
    "MaxAgeSeconds": 3600
  }]
}
```

**Option A (Recommended)**: Put CloudFront in front of S3 buckets with Response Headers Policy adding `Cross-Origin-Resource-Policy: cross-origin`

**Option B**: Use Lambda@Edge to add the header

### 1.5 Fix Image Tags

**Country flags** (`src/shared/display_flag.tsx`):
```tsx
<img src={`...${props.countryCode}.png`} crossOrigin="anonymous" />
```

**Player avatars** (`src/shared/player_avatar.tsx`):
```tsx
<Avatar src={avatarUrl} crossOrigin="anonymous" />
```

### 1.6 CSS Background Images

`src/board_modes.scss` lines 139, 157, 175, 193 reference S3 textures. If CORS issues arise:
- Move textures to same-origin (`/assets/textures/`)
- Or serve via CloudFront with proper headers

---

## Phase 2: Magpie WASM Production Packaging

### 2.1 Package Structure

Create `liwords-ui/magpie-wasm-pkg/`:

```
magpie-wasm-pkg/
├── package.json
├── magpie_wasm.js         # ES module entry
├── magpie_wasm.d.ts       # TypeScript types
├── magpie_wasm.mjs        # Emscripten module
├── magpie_wasm.wasm       # WASM binary
└── magpie_wasm.worker.js  # pthread worker
```

### 2.2 TypeScript Declarations

```typescript
export interface SimulationResult {
  moves: SimulatedMove[];
}

export interface SimulatedMove {
  position: string;    // "8G"
  word: string;        // "ZINE"
  score: number;
  equity: number;
  winPct: number;
  iterations: number;
}

export interface EndgameResult {
  bestMove: string;
  spread: number;
  pv: string[];        // Principal variation
}
```

### 2.3 Data Files

Host in `public/wasm/magpie/`:
- `winpct.csv` - Win percentages
- `english.csv` - Letter distribution
- `layouts/standard15.txt` - Board layout

Note: Reuse existing `.kwg` and `.klv2` files from wolges in `public/wasm/2024/`

---

## Phase 3: Frontend Integration

### 3.1 New Module Structure

```
liwords-ui/src/magpie/
├── MagpieContext.tsx     # React context
├── MagpieLoader.tsx      # WASM/data loading
├── useMagpie.ts          # Hook for components
├── commands.ts           # CGP string builders
├── parseResults.ts       # Parse magpie output
└── types.ts              # TypeScript interfaces
```

### 3.2 MagpieContext API

```typescript
interface MagpieContextValue {
  isAvailable: boolean;           // SharedArrayBuffer check
  isInitialized: boolean;         // WASM ready
  isRunning: boolean;             // Analysis in progress
  progress: SimProgress | null;

  initialize: (lexicon: string) => Promise<void>;
  runSimulation: (cgp: string, options: SimOptions) => Promise<SimulationResult>;
  solveEndgame: (cgp: string) => Promise<EndgameResult>;
  stop: () => void;
}
```

### 3.3 Lazy Loading

1. User clicks "Simulate" button
2. Check `SharedArrayBuffer` availability
3. Dynamically import magpie-wasm module
4. Precache lexicon files if not cached
5. Run analysis

---

## Phase 4: UI Design (Option D - Contextual)

Add "Simulate" button to Analyzer header. After quick analysis, user can run simulation to add win% column.

```
┌─────────────────────────────────────┐
│ Analyzer            [Simulate] 💡 Auto │
├─────────────────────────────────────┤
│ 8G  ZINE     32    0.0    54.2%    │
│ 7H  ZONE     26   -2.1    51.8%    │
└─────────────────────────────────────┘
```

**Endgame Mode**: When in endgame (≤7 tiles in bag), "Simulate" becomes "Solve Endgame" and runs endgame solver instead.

**File to modify**: `liwords-ui/src/gameroom/analyzer.tsx`

---

## Phase 5: Verification

### Local Testing

1. Build magpie WASM: `make -f Makefile-wasm`
2. Serve liwords with COOP/COEP headers:
   ```bash
   # In liwords-ui, modify vite.config.ts to add headers
   server: {
     headers: {
       'Cross-Origin-Opener-Policy': 'same-origin',
       'Cross-Origin-Embedder-Policy': 'require-corp',
     }
   }
   ```
3. Verify `SharedArrayBuffer` is available: `typeof SharedArrayBuffer !== 'undefined'`
4. Test fonts, avatars, flags still load

### Integration Testing

1. Run quick analysis (wolges) - verify still works
2. Run simulation - verify win% appears
3. Test endgame solver
4. Test stop/cancel during long simulation
5. Test on Chrome, Firefox, Safari

---

## Files to Create

| File | Purpose |
|------|---------|
| `liwords-ui/src/magpie/MagpieContext.tsx` | React context |
| `liwords-ui/src/magpie/MagpieLoader.tsx` | WASM loading |
| `liwords-ui/src/magpie/useMagpie.ts` | Hook |
| `liwords-ui/src/magpie/types.ts` | TypeScript types |
| `liwords-ui/src/magpie/commands.ts` | CGP builders |
| `liwords-ui/src/magpie/parseResults.ts` | Output parsing |
| `liwords-ui/magpie-wasm-pkg/*` | WASM package |
| `liwords-ui/public/wasm/magpie/*` | Data files |

## Files to Modify

| File | Changes |
|------|---------|
| `liwords-ui/index.html` | Add `crossorigin` to fonts |
| `liwords-ui/src/shared/display_flag.tsx` | Add `crossOrigin="anonymous"` |
| `liwords-ui/src/shared/player_avatar.tsx` | Add `crossOrigin="anonymous"` |
| `liwords-ui/src/gameroom/analyzer.tsx` | Add Simulate button, win% column |
| `liwords-ui/src/store/store.tsx` | Add MagpieProvider |
| `liwords-ui/package.json` | Add magpie-wasm dependency |
| `aws/cfn/cloudfront.yaml` | Add COOP/COEP headers policy |

## AWS Infrastructure Changes

| Resource | Changes |
|----------|---------|
| CloudFront woogles.io | Add Response Headers Policy with COOP/COEP |
| S3 `woogles-prod-assets` | Add CORS configuration |
| S3 `woogles-flags` | Add CORS configuration |
| S3 `woogles-uploads` | Add CORS configuration |
| (Optional) CloudFront for S3 buckets | Add `Cross-Origin-Resource-Policy` header |

---

## Implementation Order

1. **CORS/Headers Setup** (can be done incrementally, test locally first)
   - Modify index.html with crossorigin attributes
   - Test locally with Vite headers
   - Update CloudFront once verified

2. **Magpie WASM Packaging**
   - Create package structure
   - Copy WASM files from magpie build
   - Create TypeScript declarations

3. **Frontend Integration**
   - Create MagpieContext
   - Create loader following wolges pattern
   - Wire up to Analyzer component

4. **UI Polish**
   - Add Simulate button
   - Show progress during simulation
   - Handle endgame mode
