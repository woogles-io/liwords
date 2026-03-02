// Faithful TypeScript port of macondo/shell/render_template.html
// Board lives in the XY plane; Z is "up" (toward the camera).
import * as THREE from "three";
import { OrbitControls } from "three/examples/jsm/controls/OrbitControls.js";
import { FontLoader } from "three/examples/jsm/loaders/FontLoader.js";
import { TextGeometry } from "three/examples/jsm/geometries/TextGeometry.js";
import { RGBELoader } from "three/examples/jsm/loaders/RGBELoader.js";
import type { Font } from "three/examples/jsm/loaders/FontLoader.js";
import { Board3DData } from "./types";

// ─── Color maps (verbatim from macondo) ──────────────────────────────────────

const tileColors: Record<string, { hex: number; textColor: number }> = {
  orange: { hex: 0xff6b35, textColor: 0x000000 },
  yellow: { hex: 0xffa500, textColor: 0x000000 },
  pink: { hex: 0xff69b4, textColor: 0x000000 },
  red: { hex: 0xe53935, textColor: 0xffffff },
  blue: { hex: 0x1976d2, textColor: 0xffffff },
  black: { hex: 0x2c2c2c, textColor: 0xffffff },
  white: { hex: 0xf5f5f5, textColor: 0x000000 },
};

const boardColors: Record<string, number> = {
  jade: 0x00ffbd,
  teal: 0x00897b,
  blue: 0x2196f3,
  purple: 0x9c27b0,
  green: 0x4caf50,
  yellow: 0xf0c000,
  black: 0x1a1a1a,
};

const bonusColors: Record<string, number> = {
  "=": 0xcc5555,
  "-": 0xff9999,
  '"': 0x5566cc,
  "'": 0x4eb7e1,
  "~": 0x22ff22,
  "^": 0x99ff99,
  "*": 0x000000,
  " ": 0xffffff,
};

const bonusLabels: Record<string, string> = {
  "=": "3W",
  "-": "2W",
  '"': "3L",
  "'": "2L",
  "~": "4W",
  "^": "4L",
  "*": "",
  " ": "",
};

// Droid Sans supports accented characters (Ñ, Ł, etc.), matching macondo's font choice.
const FONT_URL =
  "https://threejs.org/examples/fonts/droid/droid_sans_regular.typeface.json";
const HDR_URL =
  "https://dl.polyhaven.org/file/ph-assets/HDRIs/hdr/1k/studio_small_09_1k.hdr";

// ─── Helpers ─────────────────────────────────────────────────────────────────

function getLabelColor(boardColor: number): number {
  const r = (boardColor >> 16) & 0xff;
  const g = (boardColor >> 8) & 0xff;
  const b = boardColor & 0xff;
  const brightness = r * 0.299 + g * 0.587 + b * 0.114;
  return brightness > 128 ? 0x222222 : 0xffffff;
}

function getLetterScore(
  letter: string,
  alphabetScores: Record<string, number>,
): number {
  if (letter === letter.toLowerCase() && letter !== letter.toUpperCase()) {
    return 0; // blank tile
  }
  const clean = letter.toUpperCase().replace(/[\[\]]/g, "");
  return alphabetScores[clean] ?? 0;
}

// three@0.160 TextGeometry uses 'height' for extrusion depth at runtime, but
// @types/three (0.183+) renamed it to 'depth'. Passing 'depth' is silently
// ignored and the runtime defaults to 50. We use 'as any' to pass 'height'.
// NOTE: ExtrudeGeometry correctly uses 'depth' — only TextGeometry is affected.
function makeTextGeo(
  text: string,
  font: Font,
  size: number,
  extrusion: number,
  curveSegments = 8,
): TextGeometry {
  return new TextGeometry(text, {
    font,
    size,
    height: extrusion,
    curveSegments,
    bevelEnabled: false,
  } as any);
}

function rackGeomParams(rackHeight: number, rackDepth: number) {
  const height1 = rackHeight * 0.4;
  const height2 = rackHeight * 0.3;
  const depth1 = 0.16 * rackDepth;
  const depth2 = 0.4 * rackDepth;
  const depth3 = 0.8 * rackDepth;
  const radius1 = 0.015 * rackDepth;
  const radius2 = 0.16 * rackDepth;
  const slope = (rackHeight - radius1 - height2) / (depth2 + radius1 - depth3);
  return { height1, height2, depth1, depth2, depth3, radius1, radius2, slope };
}

// Creates a single tile Group (rounded-rect ExtrudeGeometry + letter + score).
// The shape origin is (0, 0, 0); it extrudes to z = tileDepth.
// Letter/score glyphs are placed at z = tileDepth (front face, facing +Z / camera).
function createTile(
  letter: string,
  score: number,
  font: Font,
  tileColorConfig: { hex: number; textColor: number },
  tileColor: string,
  squareSize: number,
  tileDepth: number,
): THREE.Group {
  const group = new THREE.Group();
  const width = squareSize - 0.75;
  const height = squareSize - 0.25;
  const radius = 0.5;

  // Rounded-rectangle shape (verbatim from macondo)
  const shape = new THREE.Shape();
  shape.moveTo(radius, 0);
  shape.lineTo(width - radius, 0);
  shape.quadraticCurveTo(width, 0, width, radius);
  shape.lineTo(width, height - radius);
  shape.quadraticCurveTo(width, height, width - radius, height);
  shape.lineTo(radius, height);
  shape.quadraticCurveTo(0, height, 0, height - radius);
  shape.lineTo(0, radius);
  shape.quadraticCurveTo(0, 0, radius, 0);

  const geometry = new THREE.ExtrudeGeometry(shape, {
    steps: 1,
    depth: tileDepth,
    bevelEnabled: false,
  });
  const material = new THREE.MeshStandardMaterial({
    color: tileColorConfig.hex,
    roughness: 0.7,
    metalness: 0.1,
    envMapIntensity: 1.5,
  });
  const tile = new THREE.Mesh(geometry, material);
  tile.castShadow = true;
  tile.receiveShadow = true;
  group.add(tile);

  // Letter
  const displayLetter = letter.toUpperCase().replace(/[\[\]]/g, "");
  let fontSize: number;
  if (displayLetter.length === 1) {
    fontSize = Math.min(2.6, width * 0.6);
  } else if (displayLetter.length === 2) {
    fontSize = Math.min(2.0, width * 0.45);
  } else {
    fontSize = Math.min(1.6, width * 0.35);
  }

  const letterGeometry = makeTextGeo(displayLetter, font, fontSize, 0.1, 12);

  const isBlank =
    letter === letter.toLowerCase() && letter !== letter.toUpperCase();
  let textColor = tileColorConfig.textColor;
  if (isBlank) {
    textColor =
      tileColor === "red" || tileColor === "pink" ? 0x0000ff : 0xff0000;
  }

  const letterMaterial = new THREE.MeshBasicMaterial({ color: textColor });
  const letterMesh = new THREE.Mesh(letterGeometry, letterMaterial);
  letterMesh.castShadow = true;

  let xOffset: number;
  if (displayLetter.length >= 2) {
    xOffset = 0.05;
  } else if (displayLetter === "I") {
    xOffset = 0.32;
  } else if (displayLetter === "J") {
    xOffset = 0.25;
  } else if (displayLetter === "W" || displayLetter === "M") {
    xOffset = 0.1;
  } else {
    xOffset = 0.15;
  }
  letterMesh.position.set(xOffset * width, 0.2 * height, tileDepth);
  group.add(letterMesh);

  // Score
  if (score > 0) {
    const scoreGeometry = makeTextGeo(score.toString(), font, squareSize * 0.2, 0.1);
    const scoreMesh = new THREE.Mesh(scoreGeometry, letterMaterial);
    scoreMesh.castShadow = true;
    const scoreXOffset = score >= 10 ? 0.62 : 0.75;
    scoreMesh.position.set(scoreXOffset * width, 0.1 * height, tileDepth);
    group.add(scoreMesh);
  }

  return group;
}

// ─── Main scene class ─────────────────────────────────────────────────────────

export class Board3DScene {
  private renderer: THREE.WebGLRenderer;
  private camera: THREE.PerspectiveCamera;
  private scene: THREE.Scene;
  private controls: OrbitControls;
  private animationId: number | null = null;
  private resizeObserver: ResizeObserver | null = null;

  constructor(
    private container: HTMLElement,
    private data: Board3DData,
  ) {
    const w = container.clientWidth;
    const h = container.clientHeight;

    // Renderer
    this.renderer = new THREE.WebGLRenderer({
      antialias: true,
      preserveDrawingBuffer: true,
    });
    this.renderer.setSize(w, h);
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    this.renderer.shadowMap.enabled = true;
    this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
    this.renderer.toneMapping = THREE.ACESFilmicToneMapping;
    this.renderer.toneMappingExposure = 1.2;
    container.appendChild(this.renderer.domElement);

    // Camera — same as macondo: looking at board (XY plane) from +Z
    this.camera = new THREE.PerspectiveCamera(50, w / h, 0.1, 1000);
    this.camera.position.set(0, -50, 130);
    this.camera.lookAt(0, -10, 0);

    // Scene
    this.scene = new THREE.Scene();
    this.scene.background = new THREE.Color(0x222222); // overwritten by HDR if loaded

    // Orbit controls
    this.controls = new OrbitControls(this.camera, this.renderer.domElement);
    this.controls.enableDamping = false;
    this.controls.target.set(0, -10, 0);
    this.controls.update();

    // Lighting (from macondo — bright ambient for clear visibility)
    const ambientLight = new THREE.AmbientLight(0xffffff, 1.2);
    this.scene.add(ambientLight);

    const dirLight1 = new THREE.DirectionalLight(0xffffff, 0.5);
    dirLight1.position.set(50, 100, 50);
    dirLight1.castShadow = true;
    dirLight1.shadow.mapSize.width = 2048;
    dirLight1.shadow.mapSize.height = 2048;
    dirLight1.shadow.camera.far = 500;
    dirLight1.shadow.camera.left = -100;
    dirLight1.shadow.camera.right = 100;
    dirLight1.shadow.camera.top = 100;
    dirLight1.shadow.camera.bottom = -100;
    dirLight1.shadow.bias = -0.0001;
    this.scene.add(dirLight1);

    const dirLight2 = new THREE.DirectionalLight(0xffffff, 0.3);
    dirLight2.position.set(-30, 50, -30);
    this.scene.add(dirLight2);

    const pointLight = new THREE.PointLight(0xfff8e7, 0.3);
    pointLight.position.set(0, 20, 30);
    this.scene.add(pointLight);

    // HDR environment (PMREMGenerator for proper reflections, matching macondo)
    const pmremGenerator = new THREE.PMREMGenerator(this.renderer);
    pmremGenerator.compileEquirectangularShader();
    new RGBELoader().load(
      HDR_URL,
      (texture) => {
        const envMap = pmremGenerator.fromEquirectangular(texture).texture;
        this.scene.environment = envMap;
        this.scene.background = envMap;
        // backgroundBlurriness/backgroundIntensity exist on THREE.Scene in r160
        (this.scene as any).backgroundBlurriness = 0.25;
        (this.scene as any).backgroundIntensity = 0.5;
        texture.dispose();
        pmremGenerator.dispose();
      },
      undefined,
      () => {
        /* HDR failed — keep solid background */
      },
    );

    this._buildScene();

    // Animation loop
    const animate = () => {
      this.animationId = requestAnimationFrame(animate);
      this.controls.update();
      this.renderer.render(this.scene, this.camera);
    };
    animate();

    // Resize handling
    this.resizeObserver = new ResizeObserver(() => {
      const nw = container.clientWidth;
      const nh = container.clientHeight;
      this.camera.aspect = nw / nh;
      this.camera.updateProjectionMatrix();
      this.renderer.setSize(nw, nh);
    });
    this.resizeObserver.observe(container);
  }

  private _buildScene() {
    const data = this.data;
    const gridSize = data.boardDimension;
    const squareSize = (5 * 15) / gridSize; // total board footprint ~75 units
    const boardThickness = 2;
    const gridHeight = 1;
    const tileDepth = (1.5 * 15) / gridSize;
    const offset = (gridSize * squareSize) / 2 - squareSize / 2;
    const boardTileZPos = boardThickness / 2 + gridHeight;

    // Rack constants (verbatim from macondo)
    const rackHeight = 3;
    const rackWidth = 50;
    const rackDepth = 7;
    const rackYPos = -38;

    const tileColorConfig = tileColors[data.tileColor] ?? tileColors["orange"];
    const boardColorHex = boardColors[data.boardColor] ?? boardColors["jade"];
    const labelColor = getLabelColor(boardColorHex);

    // ── Circular board base ───────────────────────────────────────────────────
    // CylinderGeometry axis is Y; rotating PI/2 around X makes axis Z (board in XY plane).
    const baseGeometry = new THREE.CylinderGeometry(55, 55, boardThickness, 64);
    const baseMaterial = new THREE.MeshStandardMaterial({
      color: boardColorHex,
      roughness: 0.6,
      metalness: 0.05,
      envMapIntensity: 0.8,
    });
    const base = new THREE.Mesh(baseGeometry, baseMaterial);
    base.rotation.x = Math.PI / 2;
    base.position.z = 0;
    base.receiveShadow = true;
    this.scene.add(base);

    // ── Grid squares + walls ──────────────────────────────────────────────────
    const gridBottomZPos = boardThickness / 2;
    const wallThickness = squareSize * 0.05;
    const wallHeight = squareSize * 0.11;
    const wallMat = new THREE.MeshBasicMaterial({ color: 0x444444 });

    for (let i = 0; i < gridSize; i++) {
      // i = column (X), j = visual-Y row (j=0 at bottom, j=gridSize-1 at top)
      for (let j = 0; j < gridSize; j++) {
        // Map to our data: gridLayout[row][col], row = gridSize-1-j (top=0)
        const bonusType = data.gridLayout[gridSize - 1 - j]?.[i] ?? " ";
        const color = bonusColors[bonusType] ?? 0xffffff;

        const x = i * squareSize - offset;
        const y = j * squareSize - offset;
        const z = gridBottomZPos + gridHeight / 2;

        const squareGeom = new THREE.BoxGeometry(squareSize, squareSize, gridHeight);
        const squareMat = new THREE.MeshStandardMaterial({
          color,
          roughness: 0.8,
          metalness: 0.0,
          envMapIntensity: 0.5,
        });
        const square = new THREE.Mesh(squareGeom, squareMat);
        square.position.set(x, y, z);
        square.receiveShadow = true;
        this.scene.add(square);

        // Grid walls (thin separators between squares)
        const wz = gridBottomZPos + gridHeight;
        const addWall = (gx: number, gy: number, gz: number, wx: number, wy: number, wz: number) => {
          const wall = new THREE.Mesh(new THREE.BoxGeometry(wx, wy, wallHeight), wallMat);
          wall.position.set(gx, gy, gz);
          this.scene.add(wall);
        };
        addWall(x, y + squareSize / 2, wz, squareSize, wallThickness, wallHeight);
        addWall(x, y - squareSize / 2, wz, squareSize, wallThickness, wallHeight);
        addWall(x - squareSize / 2, y, wz, wallThickness, squareSize, wallHeight);
        addWall(x + squareSize / 2, y, wz, wallThickness, squareSize, wallHeight);
      }
    }

    // ── Wood table ────────────────────────────────────────────────────────────
    const tableTop = new THREE.Mesh(
      new THREE.BoxGeometry(180, 140, 4),
      new THREE.MeshStandardMaterial({
        color: 0x8b4513,
        roughness: 0.6,
        metalness: 0.1,
        envMapIntensity: 0.7,
      }),
    );
    tableTop.position.set(0, 0, -boardThickness / 2 - 2);
    tableTop.receiveShadow = true;
    tableTop.castShadow = true;
    this.scene.add(tableTop);

    const legMaterial = new THREE.MeshStandardMaterial({
      color: 0x654321,
      roughness: 0.5,
      metalness: 0.1,
    });
    for (const [lx, ly] of [[-80, -60], [80, -60], [-80, 60], [80, 60]] as [number, number][]) {
      const leg = new THREE.Mesh(new THREE.BoxGeometry(4, 4, 40), legMaterial);
      leg.position.set(lx, ly, -boardThickness / 2 - 22);
      leg.castShadow = true;
      this.scene.add(leg);
    }

    // ── Font-dependent content (tiles, rack, labels, unseen, scorepad) ────────
    const fontLoader = new FontLoader();
    fontLoader.load(
      FONT_URL,
      (font: Font) => {
        this._buildBoardContent(
          font, gridSize, squareSize, offset, boardTileZPos, tileDepth,
          gridBottomZPos, tileColorConfig, labelColor, data,
        );
        this._buildRack(font, rackWidth, rackHeight, rackDepth, rackYPos, squareSize, tileDepth, tileColorConfig, data);
        this._buildUnseenTiles(font, squareSize, offset, boardThickness, tileDepth, tileColorConfig, data);
        this._buildScorepad(data);
      },
      undefined,
      () => {
        // Font load failed — board shows without text
      },
    );
  }

  // ── Board tiles, bonus labels, row/col labels ─────────────────────────────

  private _buildBoardContent(
    font: Font,
    gridSize: number,
    squareSize: number,
    offset: number,
    boardTileZPos: number,
    tileDepth: number,
    gridBottomZPos: number,
    tileColorConfig: { hex: number; textColor: number },
    labelColor: number,
    data: Board3DData,
  ) {
    const labelMat = new THREE.MeshBasicMaterial({ color: labelColor });

    // Bonus labels on empty squares
    for (let i = 0; i < gridSize; i++) {
      for (let j = 0; j < gridSize; j++) {
        const bonusType = data.gridLayout[gridSize - 1 - j]?.[i] ?? " ";
        const labelText = bonusLabels[bonusType];
        if (!labelText) continue;

        const x = i * squareSize - offset;
        const y = j * squareSize - offset;

        const labelGeom = makeTextGeo(labelText, font, squareSize * 0.28, 0.02, 4);
        labelGeom.computeBoundingBox();
        const bb = labelGeom.boundingBox!;
        const textWidth = bb.max.x - bb.min.x;
        const textHeight = bb.max.y - bb.min.y;
        const xNudge = labelText.endsWith("L") ? 0.15 : 0;
        const labelMesh = new THREE.Mesh(labelGeom, labelMat);
        labelMesh.position.set(
          x - textWidth / 2 + xNudge,
          y - textHeight / 2,
          gridBottomZPos + 1 + 0.01, // gridHeight = 1
        );
        this.scene.add(labelMesh);
      }
    }

    // Board tiles
    // data.boardArray[y][x]: y=0 is the top row; x=0 is column A.
    // World: posX = x*squareSize - offset - width/2 + small adj
    //        posY = (gridSize-1-y)*squareSize - offset - height/2 + small adj
    for (let y = 0; y < gridSize; y++) {
      for (let x = 0; x < gridSize; x++) {
        const tileStr = data.boardArray[y]?.[x] ?? "";
        if (!tileStr) continue;
        const score = getLetterScore(tileStr, data.alphabetScores);
        const tileGroup = createTile(tileStr, score, font, tileColorConfig, data.tileColor, squareSize, tileDepth);
        const posX = x * squareSize - offset - squareSize / 2 + 0.375;
        const posY = (gridSize - 1 - y) * squareSize - offset - squareSize / 2 + 0.125;
        tileGroup.position.set(posX, posY, boardTileZPos);
        this.scene.add(tileGroup);
      }
    }

    // Column labels (A, B, C, …)
    for (let i = 0; i < gridSize; i++) {
      const letter = String.fromCharCode(65 + i);
      const geo = makeTextGeo(letter, font, squareSize * 0.375, 0.05);
      const mesh = new THREE.Mesh(geo, labelMat);
      mesh.position.set(
        i * squareSize - offset - squareSize / 4,
        offset + squareSize * 0.8,
        1 + 0.01, // boardThickness/2 = 1
      );
      this.scene.add(mesh);
    }

    // Row labels (1-15)
    for (let i = 0; i < gridSize; i++) {
      const number = (i + 1).toString();
      const geo = makeTextGeo(number, font, squareSize * 0.375, 0.05);
      const mesh = new THREE.Mesh(geo, labelMat);
      const textWidth = number.length * squareSize * 0.3;
      mesh.position.set(
        -offset - squareSize * 0.65 - textWidth,
        (gridSize - 1 - i) * squareSize - offset - squareSize / 4,
        1 + 0.01, // boardThickness/2 = 1
      );
      this.scene.add(mesh);
    }
  }

  // ── Rack + rack tiles ─────────────────────────────────────────────────────

  private _buildRack(
    font: Font,
    rackWidth: number,
    rackHeight: number,
    rackDepth: number,
    rackYPos: number,
    squareSize: number,
    tileDepth: number,
    tileColorConfig: { hex: number; textColor: number },
    data: Board3DData,
  ) {
    // Rack body (verbatim from macondo createRack)
    const { height2, depth3, depth2, depth1, radius1, radius2 } = rackGeomParams(rackHeight, rackDepth);

    const shape = new THREE.Shape();
    shape.moveTo(radius1, 0);
    shape.lineTo(rackDepth - radius1, 0);
    shape.quadraticCurveTo(rackDepth, 0, rackDepth, radius1);
    shape.lineTo(rackDepth, height2);

    const controlPointX = (rackDepth + depth3) / 2;
    const controlPointY = height2 + radius2;
    shape.bezierCurveTo(controlPointX, controlPointY, controlPointX, height2, depth3, height2);

    shape.lineTo(depth2 + radius1, rackHeight - radius1);
    shape.quadraticCurveTo(depth2, rackHeight, depth2 - radius1, rackHeight);
    shape.lineTo(depth1 + radius1, rackHeight);
    shape.quadraticCurveTo(depth1, rackHeight, depth1 - radius1, rackHeight - radius1);
    shape.lineTo(0, height2);
    shape.lineTo(0, radius1);
    shape.quadraticCurveTo(0, 0, radius1, 0);

    const rackGeom = new THREE.ExtrudeGeometry(shape, {
      steps: 1,
      depth: rackWidth,
      bevelEnabled: false,
    });
    const rackMat = new THREE.MeshStandardMaterial({
      color: 0xc8a850,
      roughness: 0.4,
      metalness: 0.2,
      envMapIntensity: 1.0,
    });
    const rack = new THREE.Mesh(rackGeom, rackMat);
    rack.position.set(rackWidth / 2, rackYPos, 2);
    rack.rotation.set(Math.PI / 2, (3 * Math.PI) / 2, 0);
    rack.receiveShadow = true;
    rack.castShadow = true;
    this.scene.add(rack);

    // Rack tiles
    if (data.rack.length > 0) {
      const { slope } = rackGeomParams(rackHeight, rackDepth);
      const rotation = -Math.atan(slope);
      for (let i = 0; i < data.rack.length; i++) {
        const letter = data.rack[i];
        const score = getLetterScore(letter, data.alphabetScores);
        const tileGroup = createTile(letter, score, font, tileColorConfig, data.tileColor, squareSize, tileDepth);
        const xpos = -rackWidth / 2 + 2 * squareSize + i * (squareSize - 0.6);
        const ypos = rackYPos - squareSize - 0.9;
        const zpos = 1.8;
        tileGroup.position.set(xpos, ypos, zpos);
        tileGroup.rotation.x = rotation;
        this.scene.add(tileGroup);
      }
    }
  }

  // ── Unseen tile pool ──────────────────────────────────────────────────────

  private _buildUnseenTiles(
    font: Font,
    squareSize: number,
    offset: number,
    boardThickness: number,
    tileDepth: number,
    tileColorConfig: { hex: number; textColor: number },
    data: Board3DData,
  ) {
    const remaining = data.remainingTiles;
    if (!remaining || Object.keys(remaining).length === 0) return;

    const tilesPerRow = 5;
    const tileSpacing = squareSize + 0.5;
    const startX = 55 + squareSize * 0.25; // just outside the circular base (radius 55)
    const startY = offset;
    const tableTopZ = -boardThickness / 2;

    // Sort: blanks first, then alphabetically
    const sortedEntries = Object.entries(remaining).sort(([a], [b]) => {
      if (a === "?") return -1;
      if (b === "?") return 1;
      return a.localeCompare(b);
    });

    let currentIndex = 0;
    for (const [letter, count] of sortedEntries) {
      const score = getLetterScore(letter, data.alphabetScores);
      for (let k = 0; k < count; k++) {
        const row = Math.floor(currentIndex / tilesPerRow);
        const col = currentIndex % tilesPerRow;
        const x = startX + col * tileSpacing;
        const y = startY - row * tileSpacing;
        const tileGroup = createTile(letter, score, font, tileColorConfig, data.tileColor, squareSize, tileDepth);
        tileGroup.position.set(x, y, tableTopZ);
        this.scene.add(tileGroup);
        currentIndex++;
      }
    }

    // "Unseen Tiles" label above the pool
    const labelGeom = makeTextGeo("Unseen Tiles", font, 2.5, 0.1);
    const labelMesh = new THREE.Mesh(
      labelGeom,
      new THREE.MeshBasicMaterial({ color: 0x333333 }),
    );
    const labelX = startX + (tilesPerRow * tileSpacing) / 2 - 7;
    const labelY = startY + squareSize * 1.5;
    labelMesh.position.set(labelX, labelY, tableTopZ + 0.1);
    this.scene.add(labelMesh);
  }

  // ── Scorepad ──────────────────────────────────────────────────────────────

  private _buildScorepad(data: Board3DData) {
    const canvas = document.createElement("canvas");
    canvas.width = 768;
    canvas.height = 480;
    const ctx = canvas.getContext("2d")!;

    // Notepad background
    ctx.fillStyle = "#f9f3e8";
    ctx.fillRect(0, 0, canvas.width, canvas.height);

    // Lines
    ctx.strokeStyle = "#a09080";
    ctx.lineWidth = 3;
    for (let i = 0; i < 10; i++) {
      const lineY = 75 + i * 45;
      ctx.beginPath();
      ctx.moveTo(60, lineY);
      ctx.lineTo(canvas.width - 60, lineY);
      ctx.stroke();
    }

    // Red margin line
    ctx.strokeStyle = "#d32f2f";
    ctx.lineWidth = 4;
    ctx.beginPath();
    ctx.moveTo(60, 45);
    ctx.lineTo(60, canvas.height - 45);
    ctx.stroke();

    // Title
    ctx.fillStyle = "#000000";
    ctx.font = "bold 56px Arial";
    ctx.fillText("Current Scores", 90, 60);

    ctx.font = "bold 46px Arial";

    // Player 0
    ctx.fillText(data.player0Name + ":", 90, 128);
    if (data.playerOnTurn === 0) {
      ctx.fillStyle = "#DC143C";
      ctx.font = "bold 42px Arial";
      ctx.fillText("★", 630, 128);
      ctx.fillStyle = "#000000";
      ctx.font = "bold 46px Arial";
    }
    ctx.fillText(data.player0Score.toString(), 525, 128);

    // Player 1
    ctx.fillText(data.player1Name + ":", 90, 188);
    if (data.playerOnTurn === 1) {
      ctx.fillStyle = "#DC143C";
      ctx.font = "bold 42px Arial";
      ctx.fillText("★", 630, 188);
      ctx.fillStyle = "#000000";
      ctx.font = "bold 46px Arial";
    }
    ctx.fillText(data.player1Score.toString(), 525, 188);

    // Last play
    if (data.lastPlay) {
      ctx.font = "bold 42px Arial";
      ctx.fillText("Last play:", 90, 255);
      ctx.font = "bold 38px Arial";
      const maxWidth = canvas.width - 150;
      let lineY = 300;
      const words = data.lastPlay.split(" ");
      let line = "";
      for (const word of words) {
        const testLine = line + word + " ";
        if (ctx.measureText(testLine).width > maxWidth && line) {
          ctx.fillText(line, 60, lineY);
          line = word + " ";
          lineY += 38;
        } else {
          line = testLine;
        }
      }
      if (line) ctx.fillText(line, 60, lineY);
    }

    const texture = new THREE.CanvasTexture(canvas);
    texture.needsUpdate = true;

    // PlaneGeometry with aspect ratio 768:480 = 35:21.875 (from macondo)
    const padGeometry = new THREE.PlaneGeometry(35, 21.875);
    const padMaterial = new THREE.MeshBasicMaterial({
      map: texture,
      side: THREE.DoubleSide,
    });
    const scorepad = new THREE.Mesh(padGeometry, padMaterial);
    // Position to left/bottom of board, flat on the table (rotation.x = 0 means facing +Z = camera)
    scorepad.position.set(-65, -55, -0.8);
    scorepad.rotation.x = 0;
    this.scene.add(scorepad);
  }

  saveAsPNG() {
    this.renderer.render(this.scene, this.camera);
    const link = document.createElement("a");
    link.href = this.renderer.domElement.toDataURL("image/png");
    link.download = "woogles-board.png";
    link.click();
  }

  dispose() {
    if (this.animationId !== null) {
      cancelAnimationFrame(this.animationId);
      this.animationId = null;
    }
    if (this.resizeObserver) {
      this.resizeObserver.disconnect();
      this.resizeObserver = null;
    }
    this.controls.dispose();
    this.renderer.dispose();
    if (this.renderer.domElement.parentNode === this.container) {
      this.container.removeChild(this.renderer.domElement);
    }
    this.scene.traverse((obj) => {
      if (obj instanceof THREE.Mesh) {
        obj.geometry.dispose();
        if (Array.isArray(obj.material)) {
          obj.material.forEach((m) => m.dispose());
        } else {
          obj.material.dispose();
        }
      }
    });
  }
}
